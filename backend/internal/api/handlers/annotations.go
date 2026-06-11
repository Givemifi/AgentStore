package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/knowledge"
	"agentstore/internal/middleware"
	"agentstore/internal/models"
	"agentstore/internal/syslog"
	"agentstore/internal/validation"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AnnotationsHandler powers the admin data-annotation workbench: reviewing
// flagged assistant messages, scoring/labeling them, promoting ideal answers
// into the agent knowledge base, and exporting SFT training data.
type AnnotationsHandler struct {
	db        *db.MongoDB
	knowledge *knowledge.Service
	logger    *syslog.Logger
}

// NewAnnotationsHandler creates an annotations handler.
func NewAnnotationsHandler(database *db.MongoDB, svc *knowledge.Service, logger *syslog.Logger) *AnnotationsHandler {
	return &AnnotationsHandler{db: database, knowledge: svc, logger: logger}
}

// queueItem is one row in the annotation queue.
type queueItem struct {
	MessageID      string    `json:"messageId"`
	ConversationID string    `json:"conversationId"`
	AgentID        string    `json:"agentId"`
	UserQuestion   string    `json:"userQuestion"`
	AssistantReply string    `json:"assistantReply"`
	Rating         *int      `json:"rating,omitempty"`
	Comment        string    `json:"comment,omitempty"`
	Annotated      bool      `json:"annotated"`
	QualityScore   *int      `json:"qualityScore,omitempty"`
	Status         string    `json:"status,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

const annotationQueuePageSize = 20

// GetQueue lists assistant messages awaiting/under review for the tenant. By
// default negative-feedback messages surface first; filters narrow by agent,
// rating, and annotation status.
func (h *AnnotationsHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	agentID := r.URL.Query().Get("agentId")
	ratingFilter := r.URL.Query().Get("rating")     // "1" | "-1" | ""
	statusFilter := r.URL.Query().Get("status")      // "annotated" | "unannotated" | ""
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 0 {
		page = 0
	}

	// Source the queue from feedback first (most actionable), joining message text.
	feedbackFilter := bson.M{"tenantId": tenant.ID}
	if agentID != "" {
		feedbackFilter["agentId"] = agentID
	}
	if ratingFilter == "1" || ratingFilter == "-1" {
		rv, _ := strconv.Atoi(ratingFilter)
		feedbackFilter["rating"] = rv
	}

	cursor, err := h.db.MessageFeedback().Find(r.Context(), feedbackFilter,
		options.Find().
			SetSort(bson.D{{Key: "rating", Value: 1}, {Key: "createdAt", Value: -1}}). // negatives first
			SetSkip(int64(page*annotationQueuePageSize)).
			SetLimit(int64(annotationQueuePageSize)),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load queue")
		return
	}
	defer cursor.Close(r.Context())

	var feedback []models.MessageFeedback
	if err := cursor.All(r.Context(), &feedback); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode queue")
		return
	}

	items := make([]queueItem, 0, len(feedback))
	for _, fb := range feedback {
		item, ok := h.buildQueueItem(r.Context(), tenant.ID, fb.MessageID, &fb)
		if !ok {
			continue
		}
		// Apply annotation-status filter post-hoc (annotation is a separate doc).
		if statusFilter == "annotated" && !item.Annotated {
			continue
		}
		if statusFilter == "unannotated" && item.Annotated {
			continue
		}
		items = append(items, item)
	}
	respondWithJSON(w, http.StatusOK, items)
}

// buildQueueItem assembles a queue row for an assistant message, attaching the
// preceding user question, any feedback, and existing annotation state.
func (h *AnnotationsHandler) buildQueueItem(ctx context.Context, tenantID, messageID primitive.ObjectID, fb *models.MessageFeedback) (queueItem, bool) {
	var msg models.ChatMessage
	if err := h.db.ChatMessages().FindOne(ctx, bson.M{"_id": messageID, "tenantId": tenantID}).Decode(&msg); err != nil {
		return queueItem{}, false
	}

	item := queueItem{
		MessageID:      msg.ID.Hex(),
		ConversationID: msg.ConversationID.Hex(),
		AgentID:        msg.AgentID,
		AssistantReply: truncateForPreview(msg.Content, 600),
		CreatedAt:      msg.CreatedAt,
	}
	if fb != nil {
		r := fb.Rating
		item.Rating = &r
		item.Comment = fb.Comment
	}

	// Find the user question immediately preceding this assistant message.
	if q := h.precedingUserQuestion(ctx, tenantID, msg); q != "" {
		item.UserQuestion = truncateForPreview(q, 400)
	}

	// Attach annotation state if present.
	var ann models.Annotation
	if err := h.db.Annotations().FindOne(ctx, bson.M{"tenantId": tenantID, "messageId": msg.ID}).Decode(&ann); err == nil {
		item.Annotated = true
		score := ann.QualityScore
		item.QualityScore = &score
		item.Status = string(ann.Status)
	}
	return item, true
}

// precedingUserQuestion returns the text of the latest user message before the
// given assistant message in the same conversation.
func (h *AnnotationsHandler) precedingUserQuestion(ctx context.Context, tenantID primitive.ObjectID, assistant models.ChatMessage) string {
	var prev models.ChatMessage
	err := h.db.ChatMessages().FindOne(ctx,
		bson.M{
			"tenantId":       tenantID,
			"conversationId": assistant.ConversationID,
			"role":           "user",
			"createdAt":      bson.M{"$lte": assistant.CreatedAt},
		},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	).Decode(&prev)
	if err != nil {
		return ""
	}
	return prev.Content
}

func truncateForPreview(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max]) + "…"
}

// contextMessage is one message in a conversation context view.
type contextMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetContext returns the full conversation surrounding a message so an admin can
// review the exchange before annotating. This reads another user's conversation
// for quality review, scoped strictly to the admin's tenant, and is audit-logged.
func (h *AnnotationsHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	admin, _ := middleware.GetUserFromContext(r.Context())
	messageID, err := primitive.ObjectIDFromHex(mux.Vars(r)["messageId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid message ID")
		return
	}

	var msg models.ChatMessage
	if err := h.db.ChatMessages().FindOne(r.Context(), bson.M{"_id": messageID, "tenantId": tenant.ID}).Decode(&msg); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load message")
		return
	}

	cursor, err := h.db.ChatMessages().Find(r.Context(),
		bson.M{"tenantId": tenant.ID, "conversationId": msg.ConversationID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load conversation")
		return
	}
	defer cursor.Close(r.Context())

	var msgs []models.ChatMessage
	if err := cursor.All(r.Context(), &msgs); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode conversation")
		return
	}

	out := make([]contextMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, contextMessage{ID: m.ID.Hex(), Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt})
	}

	// Audit: an admin viewed another user's conversation for quality review.
	if h.logger != nil {
		h.logger.Medium(r.Context(), fmt.Sprintf("Annotation review: admin %s viewed conversation %s (message %s) in tenant %s",
			admin.ID.Hex(), msg.ConversationID.Hex(), messageID.Hex(), tenant.ID.Hex()))
	}

	respondWithJSON(w, http.StatusOK, out)
}

type upsertAnnotationRequest struct {
	QualityScore int      `json:"qualityScore"`
	IssueTags    []string `json:"issueTags"`
	IdealAnswer  string   `json:"idealAnswer"`
	Notes        string   `json:"notes"`
}

// UpsertAnnotation creates or updates the annotation for a message.
func (h *AnnotationsHandler) UpsertAnnotation(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	admin, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	messageID, err := primitive.ObjectIDFromHex(mux.Vars(r)["messageId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid message ID")
		return
	}

	var req upsertAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var msg models.ChatMessage
	if err := h.db.ChatMessages().FindOne(r.Context(), bson.M{"_id": messageID, "tenantId": tenant.ID}).Decode(&msg); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load message")
		return
	}
	if msg.Role != "assistant" {
		respondWithError(w, http.StatusBadRequest, "only assistant messages can be annotated")
		return
	}

	now := time.Now()
	tags := req.IssueTags
	if tags == nil {
		tags = []string{}
	}
	candidate := models.Annotation{
		TenantID:       tenant.ID,
		ConversationID: msg.ConversationID,
		MessageID:      messageID,
		AgentID:        msg.AgentID,
		AnnotatorID:    admin.ID,
		QualityScore:   req.QualityScore,
		IssueTags:      tags,
		IdealAnswer:    strings.TrimSpace(req.IdealAnswer),
		Notes:          strings.TrimSpace(req.Notes),
		Status:         models.AnnotationStatusAnnotated,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validation.Validate(&candidate); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.db.Annotations().UpdateOne(r.Context(),
		bson.M{"tenantId": tenant.ID, "messageId": messageID},
		bson.M{
			"$set": bson.M{
				"qualityScore": candidate.QualityScore,
				"issueTags":    candidate.IssueTags,
				"idealAnswer":  candidate.IdealAnswer,
				"notes":        candidate.Notes,
				"annotatorId":  admin.ID,
				"updatedAt":    now,
			},
			"$setOnInsert": bson.M{
				"conversationId": msg.ConversationID,
				"agentId":        msg.AgentID,
				"status":         models.AnnotationStatusAnnotated,
				"createdAt":      now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to save annotation")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "annotated"})
}

// PromoteAnnotation turns an annotation's ideal answer into a knowledge document
// for the agent (so the corrected answer immediately improves future replies).
func (h *AnnotationsHandler) PromoteAnnotation(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	admin, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	messageID, err := primitive.ObjectIDFromHex(mux.Vars(r)["messageId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid message ID")
		return
	}

	var ann models.Annotation
	if err := h.db.Annotations().FindOne(r.Context(), bson.M{"tenantId": tenant.ID, "messageId": messageID}).Decode(&ann); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "annotation not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load annotation")
		return
	}
	if ann.Status == models.AnnotationStatusPromoted {
		respondWithError(w, http.StatusConflict, "annotation already promoted")
		return
	}
	if strings.TrimSpace(ann.IdealAnswer) == "" {
		respondWithError(w, http.StatusBadRequest, "annotation has no ideal answer to promote")
		return
	}

	// Knowledge bases are only supported for DB agents owned by this tenant.
	agentObjID, err := primitive.ObjectIDFromHex(ann.AgentID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "knowledge promotion is only supported for workspace agents")
		return
	}
	var agent models.Agent
	if err := h.db.Agents().FindOne(r.Context(), bson.M{"_id": agentObjID, "tenantId": tenant.ID}).Decode(&agent); err != nil {
		respondWithError(w, http.StatusBadRequest, "agent not found in this workspace")
		return
	}

	question := h.precedingUserQuestion(r.Context(), tenant.ID, models.ChatMessage{
		ConversationID: ann.ConversationID,
		CreatedAt:      time.Now(),
	})
	text := strings.TrimSpace(ann.IdealAnswer)
	if question != "" {
		text = fmt.Sprintf("问:%s\n答:%s", strings.TrimSpace(question), strings.TrimSpace(ann.IdealAnswer))
	}

	now := time.Now()
	doc := models.KnowledgeDocument{
		ID:         primitive.NewObjectID(),
		TenantID:   tenant.ID,
		AgentID:    ann.AgentID,
		Name:       "Annotated QA " + now.Format("2006-01-02 15:04"),
		SourceType: models.KnowledgeSourceAnnotation,
		Status:     models.KnowledgeStatusProcessing,
		CharCount:  len(text),
		CreatedBy:  admin.ID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := validation.Validate(&doc); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.db.KnowledgeDocuments().InsertOne(r.Context(), doc); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create knowledge document")
		return
	}

	// Ingest synchronously so the caller learns whether embedding succeeded; this
	// is an explicit admin action, not a hot path.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ingestErr := h.knowledge.IngestDocument(ctx, *tenant, doc, text, 1000)
	if ingestErr != nil {
		_, _ = h.db.KnowledgeDocuments().UpdateOne(ctx, bson.M{"_id": doc.ID}, bson.M{"$set": bson.M{
			"status": models.KnowledgeStatusError, "errorMessage": truncateForPreview(ingestErr.Error(), 500), "updatedAt": time.Now(),
		}})
		respondWithError(w, http.StatusBadRequest, "failed to embed promoted knowledge: "+ingestErr.Error())
		return
	}
	n, _ := h.db.KnowledgeChunks().CountDocuments(ctx, bson.M{"documentId": doc.ID})
	_, _ = h.db.KnowledgeDocuments().UpdateOne(ctx, bson.M{"_id": doc.ID}, bson.M{"$set": bson.M{
		"status": models.KnowledgeStatusReady, "chunkCount": int(n), "updatedAt": time.Now(),
	}})

	_, _ = h.db.Annotations().UpdateOne(r.Context(),
		bson.M{"tenantId": tenant.ID, "messageId": messageID},
		bson.M{"$set": bson.M{"status": models.AnnotationStatusPromoted, "promotedDocumentId": doc.ID, "updatedAt": time.Now()}},
	)
	respondWithJSON(w, http.StatusOK, map[string]any{"status": "promoted", "documentId": doc.ID.Hex(), "chunkCount": n})
}

// ExportTraining streams an SFT JSONL dataset for an agent. Each line is a chat
// sample: system prompt + conversation turns up to the annotated message, with
// the assistant turn replaced by the ideal answer (or the original answer when
// the score is high and no ideal answer was written).
func (h *AnnotationsHandler) ExportTraining(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	agentID := r.URL.Query().Get("agentId")
	minScore, _ := strconv.Atoi(r.URL.Query().Get("minScore"))
	if minScore <= 0 {
		minScore = 4
	}

	filter := bson.M{"tenantId": tenant.ID, "qualityScore": bson.M{"$gte": minScore}}
	if agentID != "" {
		filter["agentId"] = agentID
	}
	cursor, err := h.db.Annotations().Find(r.Context(), filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load annotations")
		return
	}
	defer cursor.Close(r.Context())

	var anns []models.Annotation
	if err := cursor.All(r.Context(), &anns); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode annotations")
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", "attachment; filename=agentstore-sft-export.jsonl")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	for _, ann := range anns {
		sample := h.buildTrainingSample(r.Context(), tenant.ID, ann)
		if sample == nil {
			continue
		}
		_ = enc.Encode(sample)
	}
}

type sftMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type sftSample struct {
	Messages []sftMessage   `json:"messages"`
	Meta     map[string]any `json:"meta"`
}

// buildTrainingSample reconstructs a single SFT example from an annotation.
func (h *AnnotationsHandler) buildTrainingSample(ctx context.Context, tenantID primitive.ObjectID, ann models.Annotation) *sftSample {
	var target models.ChatMessage
	if err := h.db.ChatMessages().FindOne(ctx, bson.M{"_id": ann.MessageID, "tenantId": tenantID}).Decode(&target); err != nil {
		return nil
	}

	// Prefer the corrected ideal answer; otherwise use the original reply.
	finalAnswer := strings.TrimSpace(ann.IdealAnswer)
	if finalAnswer == "" {
		finalAnswer = strings.TrimSpace(target.Content)
	}
	if finalAnswer == "" {
		return nil
	}

	// Resolve the agent's system prompt (DB agent preferred).
	systemPrompt := h.agentSystemPrompt(ctx, tenantID, ann.AgentID)

	msgs := []sftMessage{}
	if systemPrompt != "" {
		msgs = append(msgs, sftMessage{Role: "system", Content: systemPrompt})
	}

	// Conversation turns strictly before the target assistant message.
	cursor, err := h.db.ChatMessages().Find(ctx,
		bson.M{"tenantId": tenantID, "conversationId": ann.ConversationID, "createdAt": bson.M{"$lt": target.CreatedAt}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err == nil {
		var prior []models.ChatMessage
		if cursor.All(ctx, &prior) == nil {
			for _, m := range prior {
				if strings.TrimSpace(m.Content) == "" {
					continue
				}
				msgs = append(msgs, sftMessage{Role: m.Role, Content: m.Content})
			}
		}
		cursor.Close(ctx)
	}

	msgs = append(msgs, sftMessage{Role: "assistant", Content: finalAnswer})

	return &sftSample{
		Messages: msgs,
		Meta: map[string]any{
			"agentId":      ann.AgentID,
			"qualityScore": ann.QualityScore,
			"issueTags":    ann.IssueTags,
		},
	}
}

// agentSystemPrompt resolves an agent's system prompt from the DB (workspace or
// platform agent) or the static catalog, returning "" if unavailable.
func (h *AnnotationsHandler) agentSystemPrompt(ctx context.Context, tenantID primitive.ObjectID, agentID string) string {
	if objID, err := primitive.ObjectIDFromHex(agentID); err == nil {
		var agent models.Agent
		if h.db.Agents().FindOne(ctx, bson.M{"_id": objID}).Decode(&agent) == nil {
			return agent.SystemPrompt
		}
	}
	return ""
}

// agentStat is the quality dashboard summary for one agent.
type agentStat struct {
	AgentID          string  `json:"agentId"`
	MessageCount     int64   `json:"messageCount"`
	ThumbsUp         int64   `json:"thumbsUp"`
	ThumbsDown       int64   `json:"thumbsDown"`
	AnnotationCount  int64   `json:"annotationCount"`
	PromotedCount    int64   `json:"promotedCount"`
	AvgQualityScore  float64 `json:"avgQualityScore"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
}

// GetStats returns per-agent quality metrics for the tenant. When agentId is
// provided, only that agent's row is returned.
func (h *AnnotationsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	agentID := r.URL.Query().Get("agentId")

	match := bson.M{"tenantId": tenant.ID, "role": "assistant"}
	if agentID != "" {
		match["agentId"] = agentID
	}

	stats := map[string]*agentStat{}
	get := func(id string) *agentStat {
		if s, ok := stats[id]; ok {
			return s
		}
		s := &agentStat{AgentID: id}
		stats[id] = s
		return s
	}

	// Messages + token totals via aggregation.
	msgPipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":              "$agentId",
			"messageCount":     bson.M{"$sum": 1},
			"promptTokens":     bson.M{"$sum": "$promptTokens"},
			"completionTokens": bson.M{"$sum": "$completionTokens"},
		}}},
	}
	if cur, err := h.db.ChatMessages().Aggregate(r.Context(), msgPipeline); err == nil {
		var rows []struct {
			ID               string `bson:"_id"`
			MessageCount     int64  `bson:"messageCount"`
			PromptTokens     int64  `bson:"promptTokens"`
			CompletionTokens int64  `bson:"completionTokens"`
		}
		if cur.All(r.Context(), &rows) == nil {
			for _, row := range rows {
				s := get(row.ID)
				s.MessageCount = row.MessageCount
				s.PromptTokens = row.PromptTokens
				s.CompletionTokens = row.CompletionTokens
			}
		}
		cur.Close(r.Context())
	}

	// Feedback up/down counts.
	fbMatch := bson.M{"tenantId": tenant.ID}
	if agentID != "" {
		fbMatch["agentId"] = agentID
	}
	fbPipeline := mongo.Pipeline{
		{{Key: "$match", Value: fbMatch}},
		{{Key: "$group", Value: bson.M{"_id": bson.M{"agentId": "$agentId", "rating": "$rating"}, "count": bson.M{"$sum": 1}}}},
	}
	if cur, err := h.db.MessageFeedback().Aggregate(r.Context(), fbPipeline); err == nil {
		var rows []struct {
			ID struct {
				AgentID string `bson:"agentId"`
				Rating  int    `bson:"rating"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}
		if cur.All(r.Context(), &rows) == nil {
			for _, row := range rows {
				s := get(row.ID.AgentID)
				if row.ID.Rating > 0 {
					s.ThumbsUp = row.Count
				} else {
					s.ThumbsDown = row.Count
				}
			}
		}
		cur.Close(r.Context())
	}

	// Annotation counts + average score + promoted count.
	annMatch := bson.M{"tenantId": tenant.ID}
	if agentID != "" {
		annMatch["agentId"] = agentID
	}
	annPipeline := mongo.Pipeline{
		{{Key: "$match", Value: annMatch}},
		{{Key: "$group", Value: bson.M{
			"_id":          "$agentId",
			"count":        bson.M{"$sum": 1},
			"avgScore":     bson.M{"$avg": "$qualityScore"},
			"promoted":     bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "promoted"}}, 1, 0}}},
		}}},
	}
	if cur, err := h.db.Annotations().Aggregate(r.Context(), annPipeline); err == nil {
		var rows []struct {
			ID       string  `bson:"_id"`
			Count    int64   `bson:"count"`
			AvgScore float64 `bson:"avgScore"`
			Promoted int64   `bson:"promoted"`
		}
		if cur.All(r.Context(), &rows) == nil {
			for _, row := range rows {
				s := get(row.ID)
				s.AnnotationCount = row.Count
				s.AvgQualityScore = row.AvgScore
				s.PromotedCount = row.Promoted
			}
		}
		cur.Close(r.Context())
	}

	out := make([]agentStat, 0, len(stats))
	for _, s := range stats {
		out = append(out, *s)
	}
	respondWithJSON(w, http.StatusOK, out)
}
