package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type indexedChunk struct {
	chunk domain.IndexChunk
	terms map[string]int
	embed []float32
}

type RepoIndex struct {
	mu    sync.RWMutex
	store map[string]map[string]*indexedChunk
}

func NewRepoIndex() *RepoIndex {
	return &RepoIndex{store: make(map[string]map[string]*indexedChunk)}
}

func (idx *RepoIndex) Sync(req domain.IndexSyncRequest, maxChunks int) (int, error) {
	if idx == nil {
		return 0, nil
	}

	ws := strings.TrimSpace(req.WorkspaceID)
	if ws == "" {
		return 0, nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	bucket, ok := idx.store[ws]
	if !ok {
		bucket = make(map[string]*indexedChunk)
		idx.store[ws] = bucket
	}

	for _, id := range req.Delete {
		id = strings.TrimSpace(id)
		if id != "" {
			delete(bucket, id)
		}
	}

	for _, c := range req.Upsert {
		id := strings.TrimSpace(c.ID)
		if id == "" || strings.TrimSpace(c.Content) == "" {
			continue
		}
		var embed []float32
		if prev, ok := bucket[id]; ok && prev.chunk.Content == c.Content {
			embed = prev.embed
		}
		bucket[id] = &indexedChunk{
			chunk: c,
			terms: tokenize(c.Content),
			embed: embed,
		}
	}

	if maxChunks > 0 && len(bucket) > maxChunks {
		return len(bucket), fmt.Errorf("превышен лимит %d chunks для workspace", maxChunks)
	}

	return len(bucket), nil
}

func (idx *RepoIndex) Search(ctx context.Context, llm *LLMRunnerService, req domain.SearchRequest) (domain.SearchResponse, error) {
	ws := strings.TrimSpace(req.WorkspaceID)
	query := strings.TrimSpace(req.Query)
	if ws == "" || query == "" {
		return domain.SearchResponse{}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "keyword"
	}

	idx.mu.RLock()
	bucket := idx.store[ws]
	chunks := make([]*indexedChunk, 0, len(bucket))
	for _, c := range bucket {
		chunks = append(chunks, c)
	}
	idx.mu.RUnlock()

	if len(chunks) == 0 {
		return domain.SearchResponse{
			Hits: []domain.SearchHit{},
			Mode: mode,
		}, nil
	}

	queryTerms := tokenize(query)
	var hits []domain.SearchHit

	switch mode {
	case "semantic":
		hits = idx.searchSemantic(ctx, llm, chunks, query, limit)
	default:
		hits = idx.searchKeyword(chunks, queryTerms, limit)
		if mode == "hybrid" && llm != nil {
			sem := idx.searchSemantic(ctx, llm, chunks, query, limit)
			hits = mergeHits(hits, sem, limit)
		}
	}

	return domain.SearchResponse{
		Hits: hits,
		Mode: mode,
	}, nil
}

func (idx *RepoIndex) searchKeyword(chunks []*indexedChunk, queryTerms map[string]int, limit int) []domain.SearchHit {
	type scored struct {
		chunk *indexedChunk
		score float64
	}
	var list []scored

	for _, ch := range chunks {
		score := bm25Score(queryTerms, ch.terms, len(ch.chunk.Content))
		if score <= 0 {
			continue
		}

		list = append(list, scored{
			chunk: ch,
			score: score,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].score > list[j].score
	})
	if len(list) > limit {
		list = list[:limit]
	}

	out := make([]domain.SearchHit, 0, len(list))
	for _, s := range list {
		out = append(out, hitFromChunk(s.chunk, s.score))
	}
	return out
}

func (idx *RepoIndex) searchSemantic(ctx context.Context, llm *LLMRunnerService, chunks []*indexedChunk, query string, limit int) []domain.SearchHit {
	if llm == nil {
		return nil
	}

	qEmb, err := llm.Embed(ctx, query)
	if err != nil || len(qEmb) == 0 {
		return nil
	}

	type scored struct {
		chunk *indexedChunk
		score float64
	}
	var list []scored

	for _, ch := range chunks {
		emb, err := ch.embedding(ctx, llm)
		if err != nil || len(emb) == 0 {
			continue
		}

		sim := cosineSimilarity(qEmb, emb)
		if sim <= 0.1 {
			continue
		}

		list = append(list, scored{chunk: ch, score: sim})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].score > list[j].score
	})
	if len(list) > limit {
		list = list[:limit]
	}

	out := make([]domain.SearchHit, 0, len(list))
	for _, s := range list {
		out = append(out, hitFromChunk(s.chunk, s.score))
	}

	return out
}

func (ch *indexedChunk) embedding(ctx context.Context, llm *LLMRunnerService) ([]float32, error) {
	if ch == nil {
		return nil, fmt.Errorf("chunk is nil")
	}

	if len(ch.embed) > 0 {
		return ch.embed, nil
	}

	if llm == nil {
		return nil, fmt.Errorf("llm is nil")
	}

	emb, err := llm.Embed(ctx, ch.chunk.Content)
	if err != nil {
		return nil, err
	}

	ch.embed = emb

	return emb, nil
}

func hitFromChunk(ch *indexedChunk, score float64) domain.SearchHit {
	snippet := strings.TrimSpace(ch.chunk.Content)
	if len(snippet) > 400 {
		snippet = snippet[:400] + "..."
	}

	return domain.SearchHit{
		ID:       ch.chunk.ID,
		Path:     ch.chunk.Path,
		Language: ch.chunk.Language,
		Symbol:   ch.chunk.Symbol,
		Score:    score,
		Snippet:  snippet,
	}
}

func mergeHits(a, b []domain.SearchHit, limit int) []domain.SearchHit {
	seen := make(map[string]domain.SearchHit)
	for _, h := range a {
		seen[h.ID] = h
	}

	for _, h := range b {
		if prev, ok := seen[h.ID]; ok {
			if h.Score > prev.Score {
				seen[h.ID] = h
			}
			continue
		}
		seen[h.ID] = h
	}

	out := make([]domain.SearchHit, 0, len(seen))
	for _, h := range seen {
		out = append(out, h)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}

	return out
}

func tokenize(text string) map[string]int {
	out := make(map[string]int)
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(w) < 2 {
			continue
		}
		out[w]++
	}

	return out
}

func bm25Score(query, doc map[string]int, docLen int) float64 {
	if len(query) == 0 {
		return 0
	}

	k1, b := 1.2, 0.75
	avgDL := 500.0
	if docLen <= 0 {
		docLen = 1
	}

	var score float64
	for term, qf := range query {
		tf := float64(doc[term])
		if tf == 0 {
			continue
		}

		idf := 1.0
		num := tf * (k1 + 1)
		den := tf + k1*(1-b+b*float64(docLen)/avgDL)
		score += idf * (num / den) * float64(qf)
	}

	return score
}

func cosineSimilarity(a, b []float32) float64 {
	n := min(len(a), len(b))
	if n == 0 {
		return 0
	}

	var dot, na, nb float64
	for i := range n {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}

	if na == 0 || nb == 0 {
		return 0
	}

	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
