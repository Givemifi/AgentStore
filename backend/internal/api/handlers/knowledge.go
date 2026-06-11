package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/knowledge"
	"agentstore/internal/llm"
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

// Knowledge-base limits per agent. These bound storage and embedding cost while
// staying generous enough for real domain corpora.
const (
	maxKnowledgeDocsPerAgent   = 20
	maxKnowledgeChunksPerAgent = 1000
	maxKnowledgeTextChars      = 400_000
)

// KnowledgeHandler manages an agent's knowledge documents (admin-only writes).
type KnowledgeHandler struct {
	db        *db.MongoDB
	knowledge *knowledge.Service
	router    *llm.Router
	logger    *syslog.Logger
}

// NewKnowledgeHandler creates a knowledge handler.
func NewKnowledgeHandler(database *db.MongoDB, svc *knowledge.Service, router *llm.Router, logger *syslog.Logger) *KnowledgeHandler {
	return &KnowledgeHandler{db: database, knowledge: svc, router: router, logger: logger}
}

// resolveOwnedAgent loads a DB agent that belongs to the requesting tenant. Only
// DB agents support knowledge bases (static catalog agents are demo fallbacks).
func (h *KnowledgeHandler) resolveOwnedAgent(ctx context.Context, agentIDStr string, tenantID primitive.ObjectID) (*models.Agent, error) {
	agentID, err := primitive.ObjectIDFromHex(agentIDStr)
	if err != nil {
		return nil, errInvalidAgent
	}
	var agent models.Agent
	if err := h.db.Agents().FindOne(ctx, bson.M{"_id": agentID, "tenantId": tenantID}).Decode(&agent); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errAgentNotFound
		}
		return nil, err
	}
	return &agent, nil
}

var (
	errInvalidAgent  = errors.New("invalid agent ID")
	errAgentNotFound = errors.New("agent not found")
)

// ListKnowledge returns all knowledge documents for an agent (tenant-scoped).
func (h *KnowledgeHandler) ListKnowledge(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	agentID := mux.Vars(r)["agentId"]
	if _, err := h.resolveOwnedAgent(r.Context(), agentID, tenant.ID); err != nil {
		h.writeAgentErr(w, err)
		return
	}

	cursor, err := h.db.KnowledgeDocuments().Find(r.Context(),
		bson.M{"tenantId": tenant.ID, "agentId": agentID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list knowledge")
		return
	}
	defer cursor.Close(r.Context())

	var docs []models.KnowledgeDocument
	if err := cursor.All(r.Context(), &docs); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode knowledge")
		return
	}
	if docs == nil {
		docs = []models.KnowledgeDocument{}
	}
	respondWithJSON(w, http.StatusOK, docs)
}

type createKnowledgeRequest struct {
	Name       string `json:"name"`
	SourceType string `json:"sourceType"`
	Text       string `json:"text"`
}

// CreateKnowledge ingests a new knowledge document. The text is already extracted
// client-side (PDF/Word/TXT) per the project's "extract in browser" decision.
func (h *KnowledgeHandler) CreateKnowledge(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	agentID := mux.Vars(r)["agentId"]
	agent, err := h.resolveOwnedAgent(r.Context(), agentID, tenant.ID)
	if err != nil {
		h.writeAgentErr(w, err)
		return
	}

	var req createKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Text = strings.TrimSpace(req.Text)
	if req.Name == "" {
		req.Name = "Untitled"
	}
	if req.Text == "" {
		respondWithError(w, http.StatusBadRequest, "text is required")
		return
	}
	if len(req.Text) > maxKnowledgeTextChars {
		respondWithError(w, http.StatusBadRequest, "text exceeds size limit; please split it into smaller documents")
		return
	}
	sourceType := models.KnowledgeSourceType(req.SourceType)
	if !models.ValidKnowledgeSourceType(sourceType) {
		sourceType = models.KnowledgeSourceText
	}

	// Require an embedding model to be configured before accepting documents.
	if _, err := h.router.ResolveEmbeddingModel(r.Context(), *tenant); err != nil {
		respondWithError(w, http.StatusBadRequest, "No embedding model is configured. Set a default embedding model in Settings → Models first.")
		return
	}

	// Enforce per-agent document count.
	count, err := h.db.KnowledgeDocuments().CountDocuments(r.Context(), bson.M{"tenantId": tenant.ID, "agentId": agentID})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to count documents")
		return
	}
	if count >= maxKnowledgeDocsPerAgent {
		respondWithError(w, http.StatusBadRequest, "this agent has reached the maximum number of knowledge documents")
		return
	}

	now := time.Now()
	doc := models.KnowledgeDocument{
		ID:         primitive.NewObjectID(),
		TenantID:   tenant.ID,
		AgentID:    agentID,
		Name:       req.Name,
		SourceType: sourceType,
		Status:     models.KnowledgeStatusProcessing,
		ChunkCount: 0,
		CharCount:  len(req.Text),
		CreatedBy:  user.ID,
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

	// Ingest asynchronously: chunk + embed + store, then mark ready/error.
	go h.ingest(*tenant, doc, req.Text)

	respondWithJSON(w, http.StatusAccepted, doc)
	_ = agent // agent ownership already validated above
}

// ingest runs the embedding pipeline off the request goroutine and records the
// terminal status on the document.
func (h *KnowledgeHandler) ingest(ownerTenant models.Tenant, doc models.KnowledgeDocument, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := h.knowledge.IngestDocument(ctx, ownerTenant, doc, text, maxKnowledgeChunksPerAgent)
	update := bson.M{"updatedAt": time.Now()}
	if err != nil {
		update["status"] = models.KnowledgeStatusError
		msg := err.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		update["errorMessage"] = msg
		if h.logger != nil {
			h.logger.Medium(ctx, "Knowledge ingestion failed for document "+doc.ID.Hex()+" (agent "+doc.AgentID+"): "+msg)
		}
	} else {
		// Count the chunks we just wrote for display.
		n, _ := h.db.KnowledgeChunks().CountDocuments(ctx, bson.M{"documentId": doc.ID})
		update["status"] = models.KnowledgeStatusReady
		update["chunkCount"] = int(n)
		update["errorMessage"] = ""
	}
	_, _ = h.db.KnowledgeDocuments().UpdateOne(ctx, bson.M{"_id": doc.ID}, bson.M{"$set": update})
}

// DeleteKnowledge removes a document and its chunks, then invalidates the cache.
func (h *KnowledgeHandler) DeleteKnowledge(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	agentID := mux.Vars(r)["agentId"]
	docID, err := primitive.ObjectIDFromHex(mux.Vars(r)["docId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	result, err := h.db.KnowledgeDocuments().DeleteOne(r.Context(), bson.M{"_id": docID, "tenantId": tenant.ID, "agentId": agentID})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to delete document")
		return
	}
	if result.DeletedCount == 0 {
		respondWithError(w, http.StatusNotFound, "document not found")
		return
	}
	if _, err := h.db.KnowledgeChunks().DeleteMany(r.Context(), bson.M{"documentId": docID}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to delete chunks")
		return
	}
	h.knowledge.InvalidateAgent(tenant.ID, agentID)
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ReindexKnowledge re-embeds a document's text. Because chunk text is stored, we
// re-embed from the existing chunks' concatenated text via re-ingestion.
func (h *KnowledgeHandler) ReindexKnowledge(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	agentID := mux.Vars(r)["agentId"]
	docID, err := primitive.ObjectIDFromHex(mux.Vars(r)["docId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	var doc models.KnowledgeDocument
	if err := h.db.KnowledgeDocuments().FindOne(r.Context(), bson.M{"_id": docID, "tenantId": tenant.ID, "agentId": agentID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "document not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load document")
		return
	}

	// Reconstruct text from existing chunks (ordered by seq).
	cursor, err := h.db.KnowledgeChunks().Find(r.Context(),
		bson.M{"documentId": docID},
		options.Find().SetSort(bson.D{{Key: "seq", Value: 1}}),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load chunks")
		return
	}
	defer cursor.Close(r.Context())
	var chunks []models.KnowledgeChunk
	if err := cursor.All(r.Context(), &chunks); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode chunks")
		return
	}
	if len(chunks) == 0 {
		respondWithError(w, http.StatusBadRequest, "no chunk text to reindex")
		return
	}
	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString(c.Text)
		sb.WriteString("\n\n")
	}

	_, _ = h.db.KnowledgeDocuments().UpdateOne(r.Context(), bson.M{"_id": docID}, bson.M{"$set": bson.M{
		"status": models.KnowledgeStatusProcessing, "updatedAt": time.Now(),
	}})
	go h.ingest(*tenant, doc, sb.String())
	respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "processing"})
}

func (h *KnowledgeHandler) writeAgentErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidAgent):
		respondWithError(w, http.StatusBadRequest, "invalid agent ID")
	case errors.Is(err, errAgentNotFound):
		respondWithError(w, http.StatusNotFound, "agent not found")
	default:
		respondWithError(w, http.StatusInternalServerError, "failed to load agent")
	}
}
