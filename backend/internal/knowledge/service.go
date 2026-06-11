// Package knowledge implements a lightweight retrieval-augmented-generation
// (RAG) layer for agents: it chunks documents, embeds them with an
// OpenAI-compatible embeddings model, stores the vectors in MongoDB, and
// retrieves the most relevant chunks for a chat query using in-process cosine
// similarity.
//
// The deployment target is a self-hosted standalone mongo:7 (no Atlas Vector
// Search), so retrieval loads a tenant+agent's chunks into memory and ranks
// them with cosine similarity. A small TTL cache keeps the hot set in memory;
// it is invalidated whenever documents for that agent change.
package knowledge

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/llm"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// targetChunkChars is the approximate size of each text chunk before
	// embedding. ~1600 chars keeps chunks semantically coherent while staying
	// well under typical embedding input limits.
	targetChunkChars = 1600
	// chunkOverlapChars is how much trailing text from one chunk is repeated at
	// the start of the next, so a fact spanning a boundary is still retrievable.
	chunkOverlapChars = 200
	// maxChunkChars hard-caps an individual chunk (defensive against a single
	// gigantic paragraph with no breaks).
	maxChunkChars = 8000

	// retrieveTopK is how many chunks at most are injected into a prompt.
	retrieveTopK = 6
	// minSimilarity filters out weakly-related chunks so irrelevant knowledge is
	// never injected (which would just waste tokens and can mislead the model).
	minSimilarity = 0.35
	// maxInjectedChars bounds the total injected knowledge text.
	maxInjectedChars = 8000

	// cacheTTL is how long a tenant+agent's chunk set stays cached in memory.
	cacheTTL = 5 * time.Minute
)

// embedder is the subset of *llm.Client the service needs (for testability).
type embedder interface {
	EmbedWithConfig(ctx context.Context, config llm.RequestConfig, texts []string) ([][]float64, error)
}

// modelResolver resolves the embedding model config for a tenant (subset of *llm.Router).
type modelResolver interface {
	ResolveEmbeddingModel(ctx context.Context, tenant models.Tenant) (llm.RequestConfig, error)
}

// Service provides knowledge ingestion and retrieval.
type Service struct {
	db       *db.MongoDB
	embedder embedder
	router   modelResolver

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	chunks    []models.KnowledgeChunk
	expiresAt time.Time
}

// NewService creates a knowledge service.
func NewService(database *db.MongoDB, client *llm.Client, router *llm.Router) *Service {
	return &Service{
		db:       database,
		embedder: client,
		router:   router,
		cache:    make(map[string]cacheEntry),
	}
}

// RetrievedChunk is a chunk paired with its similarity score for a query.
type RetrievedChunk struct {
	Text  string
	Score float64
}

func cacheKey(tenantID primitive.ObjectID, agentID string) string {
	return tenantID.Hex() + ":" + agentID
}

// InvalidateAgent drops any cached chunks for a tenant+agent. Call after a
// document is added, reindexed, or deleted so the next retrieval reloads.
func (s *Service) InvalidateAgent(tenantID primitive.ObjectID, agentID string) {
	s.mu.Lock()
	delete(s.cache, cacheKey(tenantID, agentID))
	s.mu.Unlock()
}

// Chunk splits text into overlapping chunks of approximately targetChunkChars,
// breaking on paragraph and sentence boundaries where possible.
func Chunk(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Split into paragraphs first, then greedily accumulate into chunks.
	paragraphs := splitParagraphs(text)

	var chunks []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		chunks = append(chunks, strings.TrimSpace(cur.String()))
		cur.Reset()
	}

	for _, p := range paragraphs {
		// A single paragraph larger than the cap is hard-split.
		if len(p) > maxChunkChars {
			flush()
			for len(p) > maxChunkChars {
				chunks = append(chunks, strings.TrimSpace(p[:maxChunkChars]))
				p = p[maxChunkChars:]
			}
			if strings.TrimSpace(p) != "" {
				cur.WriteString(p)
			}
			continue
		}

		if cur.Len() > 0 && cur.Len()+len(p)+2 > targetChunkChars {
			// Carry overlap from the end of the current chunk into the next.
			prev := cur.String()
			flush()
			if overlap := tailOverlap(prev); overlap != "" {
				cur.WriteString(overlap)
				cur.WriteString("\n\n")
			}
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(p)
	}
	flush()

	return chunks
}

// splitParagraphs splits on blank lines, falling back to single newlines and
// then a hard size split so no element exceeds maxChunkChars conceptually.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	var out []string
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return []string{text}
	}
	return out
}

// tailOverlap returns up to chunkOverlapChars from the end of s, trimmed to a
// word boundary so the overlap reads cleanly.
func tailOverlap(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= chunkOverlapChars {
		return s
	}
	tail := s[len(s)-chunkOverlapChars:]
	if idx := strings.IndexAny(tail, " \n\t"); idx >= 0 && idx < len(tail)-1 {
		tail = tail[idx+1:]
	}
	return strings.TrimSpace(tail)
}

// IngestDocument chunks and embeds the given text, replacing any existing chunks
// for the document, then marks the document ready (or error). The owner tenant
// supplies the embedding model configuration. maxChunks caps the number of
// chunks; exceeding it returns an error so the caller can surface a "trim the
// document" message.
func (s *Service) IngestDocument(ctx context.Context, ownerTenant models.Tenant, doc models.KnowledgeDocument, text string, maxChunks int) error {
	chunks := Chunk(text)
	if len(chunks) == 0 {
		return fmt.Errorf("no extractable text")
	}
	if maxChunks > 0 && len(chunks) > maxChunks {
		return fmt.Errorf("document too large: %d chunks exceeds limit of %d; please split or trim it", len(chunks), maxChunks)
	}

	cfg, err := s.router.ResolveEmbeddingModel(ctx, ownerTenant)
	if err != nil {
		return err
	}

	vectors, err := s.embedder.EmbedWithConfig(ctx, cfg, chunks)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}
	if len(vectors) != len(chunks) {
		return fmt.Errorf("embedding count mismatch: got %d vectors for %d chunks", len(vectors), len(chunks))
	}

	// Replace existing chunks for idempotent re-ingest/reindex.
	if _, err := s.db.KnowledgeChunks().DeleteMany(ctx, bson.M{"documentId": doc.ID}); err != nil {
		return fmt.Errorf("clear old chunks: %w", err)
	}

	now := time.Now()
	docs := make([]interface{}, 0, len(chunks))
	for i, c := range chunks {
		docs = append(docs, models.KnowledgeChunk{
			ID:         primitive.NewObjectID(),
			TenantID:   doc.TenantID,
			AgentID:    doc.AgentID,
			DocumentID: doc.ID,
			Seq:        i,
			Text:       c,
			Embedding:  vectors[i],
			CreatedAt:  now,
		})
	}
	if _, err := s.db.KnowledgeChunks().InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("insert chunks: %w", err)
	}

	s.InvalidateAgent(doc.TenantID, doc.AgentID)
	return nil
}

// Retrieve embeds the query and returns the most relevant chunks for the given
// owner tenant + agent, filtered by minSimilarity and capped by maxInjectedChars.
func (s *Service) Retrieve(ctx context.Context, ownerTenant models.Tenant, agentID, query string) ([]RetrievedChunk, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	chunks, err := s.loadChunks(ctx, ownerTenant.ID, agentID)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	cfg, err := s.router.ResolveEmbeddingModel(ctx, ownerTenant)
	if err != nil {
		return nil, err
	}
	vecs, err := s.embedder.EmbedWithConfig(ctx, cfg, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("embed query: expected 1 vector, got %d", len(vecs))
	}
	qv := vecs[0]

	scored := make([]RetrievedChunk, 0, len(chunks))
	for _, c := range chunks {
		score := cosineSimilarity(qv, c.Embedding)
		if score < minSimilarity {
			continue
		}
		scored = append(scored, RetrievedChunk{Text: c.Text, Score: score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })

	var out []RetrievedChunk
	total := 0
	for _, c := range scored {
		if len(out) >= retrieveTopK {
			break
		}
		if total+len(c.Text) > maxInjectedChars && len(out) > 0 {
			break
		}
		out = append(out, c)
		total += len(c.Text)
	}
	return out, nil
}

// loadChunks returns the agent's chunks from cache or DB.
func (s *Service) loadChunks(ctx context.Context, tenantID primitive.ObjectID, agentID string) ([]models.KnowledgeChunk, error) {
	key := cacheKey(tenantID, agentID)

	s.mu.Lock()
	if entry, ok := s.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		chunks := entry.chunks
		s.mu.Unlock()
		return chunks, nil
	}
	s.mu.Unlock()

	cursor, err := s.db.KnowledgeChunks().Find(ctx, bson.M{"tenantId": tenantID, "agentId": agentID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var chunks []models.KnowledgeChunk
	if err := cursor.All(ctx, &chunks); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cacheEntry{chunks: chunks, expiresAt: time.Now().Add(cacheTTL)}
	s.mu.Unlock()

	return chunks, nil
}

// BuildKnowledgeBlock formats retrieved chunks into a system-prompt addendum.
// Returns "" when there is nothing relevant to inject.
func BuildKnowledgeBlock(chunks []RetrievedChunk) string {
	if len(chunks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# 参考知识(Reference knowledge)\n")
	b.WriteString("以下内容来自该助手的知识库,请在相关时优先据此作答;若与问题无关则忽略,不要编造引用。\n")
	for i, c := range chunks {
		fmt.Fprintf(&b, "\n[%d] %s\n", i+1, strings.TrimSpace(c.Text))
	}
	return b.String()
}

// cosineSimilarity returns the cosine similarity of two vectors, or 0 when
// either is empty or zero-length.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
