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
	mu            sync.RWMutex
	store         map[string]map[string]*indexedChunk
	imports       map[string]map[string][]string
	importedBy    map[string]map[string][]string
	symbols       map[string]map[string]string
	searchWorkers int
	qdrant        *QdrantClient
}

func NewRepoIndex(searchWorkers int, qdrant *QdrantClient) *RepoIndex {
	if searchWorkers <= 0 {
		searchWorkers = 4
	}

	return &RepoIndex{
		store:         make(map[string]map[string]*indexedChunk),
		imports:       make(map[string]map[string][]string),
		importedBy:    make(map[string]map[string][]string),
		symbols:       make(map[string]map[string]string),
		searchWorkers: searchWorkers,
		qdrant:        qdrant,
	}
}

func (idx *RepoIndex) Sync(ctx context.Context, llm *LLMRunnerService, req domain.IndexSyncRequest, maxChunks int) (int, error) {
	if idx == nil {
		return 0, nil
	}

	ws := strings.TrimSpace(req.WorkspaceID)
	if ws == "" {
		return 0, nil
	}

	idx.mu.Lock()

	bucket, ok := idx.store[ws]
	if !ok {
		bucket = make(map[string]*indexedChunk)
		idx.store[ws] = bucket
	}

	var deletedIDs []string
	for _, id := range req.Delete {
		id = strings.TrimSpace(id)
		if id != "" {
			if ch, ok := bucket[id]; ok {
				idx.removeGraphLocked(ws, ch.chunk)
			}
			delete(bucket, id)
			deletedIDs = append(deletedIDs, id)
		}
	}

	var qdrantUpserts []domain.IndexChunk
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
		idx.updateGraphLocked(ws, c)
		qdrantUpserts = append(qdrantUpserts, c)
	}

	if maxChunks > 0 && len(bucket) > maxChunks {
		count := len(bucket)
		idx.mu.Unlock()
		return count, fmt.Errorf("превышен лимит %d chunks для workspace", maxChunks)
	}

	count := len(bucket)
	idx.mu.Unlock()

	if idx.qdrant != nil && len(deletedIDs) > 0 {
		_ = idx.qdrant.Delete(ctx, ws, deletedIDs)
	}

	if idx.qdrant != nil && llm != nil {
		for _, c := range qdrantUpserts {
			emb, err := llm.Embed(ctx, c.Content)
			if err != nil || len(emb) == 0 {
				continue
			}

			_ = idx.qdrant.Upsert(ctx, ws, c.ID, emb, map[string]any{
				"path":     c.Path,
				"language": c.Language,
				"symbol":   c.Symbol,
			})
		}
	}

	return count, nil
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
		hits = idx.searchSemantic(ctx, llm, chunks, query, limit, ws)
	default:
		hits = idx.searchKeyword(chunks, queryTerms, limit)
		if mode == "hybrid" && llm != nil {
			sem := idx.searchSemantic(ctx, llm, chunks, query, limit, ws)
			hits = mergeHits(hits, sem, limit)
		}
	}

	hits = idx.expandHitsWithGraph(ws, hits, limit)

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

func (idx *RepoIndex) searchSemantic(ctx context.Context, llm *LLMRunnerService, chunks []*indexedChunk, query string, limit int, workspaceID string) []domain.SearchHit {
	if llm == nil {
		return nil
	}

	if idx.qdrant != nil {
		if hits := idx.searchQdrant(ctx, llm, workspaceID, query, limit, chunks); len(hits) > 0 {
			return hits
		}
	}

	qEmb, err := llm.Embed(ctx, query)
	if err != nil || len(qEmb) == 0 {
		return nil
	}

	workers := idx.searchWorkers
	if workers <= 0 {
		workers = 4
	}

	type scored struct {
		chunk *indexedChunk
		score float64
	}

	results := make(chan scored, len(chunks))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, ch := range chunks {
		wg.Add(1)
		go func(ch *indexedChunk) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			emb, err := ch.embedding(ctx, llm)
			if err != nil || len(emb) == 0 {
				return
			}

			sim := cosineSimilarity(qEmb, emb)
			if sim <= 0.1 {
				return
			}
			results <- scored{chunk: ch, score: sim}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var list []scored
	for s := range results {
		list = append(list, s)
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

func (idx *RepoIndex) searchQdrant(ctx context.Context, llm *LLMRunnerService, workspaceID, query string, limit int, chunks []*indexedChunk) []domain.SearchHit {
	qEmb, err := llm.Embed(ctx, query)
	if err != nil || len(qEmb) == 0 {
		return nil
	}

	ids, scores, err := idx.qdrant.Search(ctx, workspaceID, qEmb, limit)
	if err != nil || len(ids) == 0 {
		return nil
	}

	byID := make(map[string]*indexedChunk, len(chunks))
	for _, ch := range chunks {
		byID[ch.chunk.ID] = ch
	}

	out := make([]domain.SearchHit, 0, len(ids))
	for i, id := range ids {
		ch, ok := byID[id]
		if !ok {
			continue
		}

		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}

		out = append(out, hitFromChunk(ch, score))
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

func (idx *RepoIndex) updateGraphLocked(ws string, c domain.IndexChunk) {
	path := strings.TrimSpace(c.Path)
	if path == "" {
		return
	}

	if _, ok := idx.imports[ws]; !ok {
		idx.imports[ws] = make(map[string][]string)
		idx.importedBy[ws] = make(map[string][]string)
		idx.symbols[ws] = make(map[string]string)
	}

	idx.imports[ws][path] = append([]string(nil), c.Imports...)
	for _, imp := range c.Imports {
		imp = strings.TrimSpace(imp)
		if imp == "" {
			continue
		}
		idx.importedBy[ws][imp] = appendUnique(idx.importedBy[ws][imp], path)
	}

	if sym := strings.TrimSpace(c.Symbol); sym != "" {
		idx.symbols[ws][sym] = c.ID
	}
}

func (idx *RepoIndex) removeGraphLocked(ws string, c domain.IndexChunk) {
	path := strings.TrimSpace(c.Path)
	if path == "" {
		return
	}

	if bucket, ok := idx.imports[ws]; ok {
		delete(bucket, path)
	}

	for _, imp := range c.Imports {
		imp = strings.TrimSpace(imp)
		if imp == "" {
			continue
		}
		if _, ok := idx.importedBy[ws]; ok {
			idx.importedBy[ws][imp] = removeString(idx.importedBy[ws][imp], path)
		}
	}

	if sym := strings.TrimSpace(c.Symbol); sym != "" {
		if id, ok := idx.symbols[ws][sym]; ok && id == c.ID {
			delete(idx.symbols[ws], sym)
		}
	}
}

func (idx *RepoIndex) Graph(ws, path, symbol string) domain.IndexGraphResponse {
	ws = strings.TrimSpace(ws)
	path = strings.TrimSpace(path)
	symbol = strings.TrimSpace(symbol)
	if ws == "" {
		return domain.IndexGraphResponse{}
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	resp := domain.IndexGraphResponse{Path: path, Symbol: symbol}
	if path != "" {
		resp.Imports = append([]string(nil), idx.imports[ws][path]...)
		resp.ImportedBy = append([]string(nil), idx.importedBy[ws][path]...)
	}

	if symbol != "" {
		if id := idx.symbols[ws][symbol]; id != "" {
			resp.ChunkIDs = []string{id}
		}
	}

	return resp
}

func (idx *RepoIndex) expandHitsWithGraph(ws string, hits []domain.SearchHit, limit int) []domain.SearchHit {
	if len(hits) == 0 {
		return hits
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	seen := make(map[string]struct{}, len(hits))
	for i := range hits {
		seen[hits[i].ID] = struct{}{}
		if path := strings.TrimSpace(hits[i].Path); path != "" {
			hits[i].Related = append([]string(nil), idx.imports[ws][path]...)
		}
	}

	for _, h := range hits {
		for _, rel := range h.Related {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}

			for _, ch := range idx.store[ws] {
				if strings.TrimSpace(ch.chunk.Path) != rel {
					continue
				}

				if _, ok := seen[ch.chunk.ID]; ok {
					break
				}

				seen[ch.chunk.ID] = struct{}{}
				hits = append(hits, hitFromChunk(ch, h.Score*0.6))
				if len(hits) >= limit*2 {
					sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
					return hits[:limit]
				}
				break
			}
		}
	}

	if len(hits) > limit {
		sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
		hits = hits[:limit]
	}

	return hits
}

func appendUnique(list []string, v string) []string {
	for _, s := range list {
		if s == v {
			return list
		}
	}

	return append(list, v)
}

func removeString(list []string, v string) []string {
	out := list[:0]
	for _, s := range list {
		if s != v {
			out = append(out, s)
		}
	}

	return out
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
