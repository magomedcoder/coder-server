package rag

import (
	"context"
	"sort"

	"github.com/magomedcoder/coder-server/pkg/domain"
)

const NeighborOnlyChunkScore = -1e9

type ChunkIndexFetcher interface {
	GetChunksByFileChunkIndices(ctx context.Context, sessionID int64, userID int, fileID int64, embeddingModel string, chunkIndices []int) ([]domain.DocumentRAGChunk, error)
}

func ExpandSearchHitsWithNeighbors(
	ctx context.Context,
	fetch ChunkIndexFetcher,
	sessionID int64,
	userID int,
	embeddingModel string,
	restrictFileID *int64,
	hits []domain.ScoredDocumentRAGChunk,
	neighborWindow int,
) ([]domain.ScoredDocumentRAGChunk, error) {
	if neighborWindow <= 0 || fetch == nil {
		return hits, nil
	}

	type pair struct {
		fid int64
		ix  int
	}

	primary := make(map[pair]float64)
	for _, h := range hits {
		k := pair{fid: h.FileID, ix: h.ChunkIndex}
		if s, ok := primary[k]; !ok || h.Score > s {
			primary[k] = h.Score
		}
	}

	byFile := make(map[int64]map[int]struct{})
	for k := range primary {
		if restrictFileID != nil && *restrictFileID > 0 && k.fid != *restrictFileID {
			continue
		}
		if byFile[k.fid] == nil {
			byFile[k.fid] = make(map[int]struct{})
		}
		for d := -neighborWindow; d <= neighborWindow; d++ {
			ix := k.ix + d
			if ix >= 0 {
				byFile[k.fid][ix] = struct{}{}
			}
		}
	}

	var out []domain.ScoredDocumentRAGChunk
	for fid, idxSet := range byFile {
		indices := make([]int, 0, len(idxSet))
		for ix := range idxSet {
			indices = append(indices, ix)
		}
		sort.Ints(indices)

		chunks, err := fetch.GetChunksByFileChunkIndices(ctx, sessionID, userID, fid, embeddingModel, indices)
		if err != nil {
			return nil, err
		}

		for _, ch := range chunks {
			k := pair{fid: ch.FileID, ix: ch.ChunkIndex}
			score := NeighborOnlyChunkScore
			if s, ok := primary[k]; ok {
				score = s
			}
			out = append(out, domain.ScoredDocumentRAGChunk{DocumentRAGChunk: ch, Score: score})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].FileID != out[j].FileID {
			return out[i].FileID < out[j].FileID
		}
		return out[i].ChunkIndex < out[j].ChunkIndex
	})

	return out, nil
}
