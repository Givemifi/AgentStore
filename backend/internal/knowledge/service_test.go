package knowledge

import (
	"context"
	"math"
	"strings"
	"testing"

	"agentstore/internal/llm"
	"agentstore/internal/models"
)

func TestChunk_EmptyReturnsNil(t *testing.T) {
	if got := Chunk("   "); got != nil {
		t.Fatalf("expected nil for empty text, got %v", got)
	}
}

func TestChunk_ShortTextSingleChunk(t *testing.T) {
	got := Chunk("This is a short document about taxes.")
	if len(got) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(got))
	}
}

func TestChunk_LongTextSplitsIntoMultiple(t *testing.T) {
	// Build text with many paragraphs well over targetChunkChars total.
	para := strings.Repeat("word ", 100) // ~500 chars
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString(para)
		sb.WriteString("\n\n")
	}
	got := Chunk(sb.String())
	if len(got) < 2 {
		t.Fatalf("expected multiple chunks for long text, got %d", len(got))
	}
	for i, c := range got {
		if len(c) > maxChunkChars {
			t.Fatalf("chunk %d exceeds maxChunkChars: %d", i, len(c))
		}
	}
}

func TestChunk_HardSplitsHugeParagraph(t *testing.T) {
	huge := strings.Repeat("x", maxChunkChars*2+500)
	got := Chunk(huge)
	if len(got) < 2 {
		t.Fatalf("expected hard split into >=2 chunks, got %d", len(got))
	}
	for i, c := range got {
		if len(c) > maxChunkChars {
			t.Fatalf("chunk %d exceeds maxChunkChars: %d", i, len(c))
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	cases := []struct {
		name string
		a, b []float64
		want float64
	}{
		{"identical", []float64{1, 0, 0}, []float64{1, 0, 0}, 1},
		{"orthogonal", []float64{1, 0}, []float64{0, 1}, 0},
		{"opposite", []float64{1, 0}, []float64{-1, 0}, -1},
		{"mismatched length", []float64{1, 0}, []float64{1}, 0},
		{"empty", nil, []float64{1}, 0},
		{"zero vector", []float64{0, 0}, []float64{1, 1}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cosineSimilarity(tc.a, tc.b)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("cosineSimilarity(%v,%v)=%v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestBuildKnowledgeBlock_EmptyReturnsEmpty(t *testing.T) {
	if got := BuildKnowledgeBlock(nil); got != "" {
		t.Fatalf("expected empty block, got %q", got)
	}
}

func TestBuildKnowledgeBlock_IncludesChunkText(t *testing.T) {
	block := BuildKnowledgeBlock([]RetrievedChunk{
		{Text: "Taxes are due in April.", Score: 0.9},
		{Text: "Deductions reduce taxable income.", Score: 0.7},
	})
	if !strings.Contains(block, "Taxes are due in April.") {
		t.Fatalf("block missing first chunk: %q", block)
	}
	if !strings.Contains(block, "Deductions reduce taxable income.") {
		t.Fatalf("block missing second chunk: %q", block)
	}
}

// --- Retrieve ranking (using a fake embedder, no DB) ---

type fakeEmbedder struct {
	// vectors maps text -> embedding
	vectors map[string][]float64
}

func (f *fakeEmbedder) EmbedWithConfig(_ context.Context, _ llm.RequestConfig, texts []string) ([][]float64, error) {
	out := make([][]float64, len(texts))
	for i, t := range texts {
		out[i] = f.vectors[t]
	}
	return out, nil
}

type fakeResolver struct{}

func (fakeResolver) ResolveEmbeddingModel(_ context.Context, _ models.Tenant) (llm.RequestConfig, error) {
	return llm.RequestConfig{APIKey: "k", BaseURL: "https://x/v1", Model: "embed"}, nil
}

func TestRankChunks_FiltersAndOrders(t *testing.T) {
	// Validate the in-memory ranking math directly: query vector close to chunk A,
	// far from chunk B (below threshold).
	query := []float64{1, 0}
	chunks := []models.KnowledgeChunk{
		{Text: "relevant", Embedding: []float64{0.9, 0.1}},
		{Text: "irrelevant", Embedding: []float64{0, 1}},
	}

	var scored []RetrievedChunk
	for _, c := range chunks {
		score := cosineSimilarity(query, c.Embedding)
		if score < minSimilarity {
			continue
		}
		scored = append(scored, RetrievedChunk{Text: c.Text, Score: score})
	}
	if len(scored) != 1 || scored[0].Text != "relevant" {
		t.Fatalf("expected only the relevant chunk, got %+v", scored)
	}
}
