package rag

import (
	"sort"
	"unicode/utf8"

	"github.com/magomedcoder/lmpkg/domain"
	"github.com/magomedcoder/lmpkg/prompt"
)

const (
	MultiFileSearchTopK   = 6
	MultiFilePerFileLimit = 3
	MultiFileTotalLimit   = 10
)

type MultiFileCandidate struct {
	FileIndex int
	Score     float64
	Chunk     domain.ScoredDocumentRAGChunk
}

func SelectMultiFileCandidates(
	candidates []MultiFileCandidate,
	totalBudget int,
	perFileBudget int,
) map[int][]domain.ScoredDocumentRAGChunk {
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	perFileCounts := make(map[int]int, len(candidates))
	perFileRunes := make(map[int]int, len(candidates))
	selected := make(map[int][]domain.ScoredDocumentRAGChunk, len(candidates))
	selectedTotal := 0
	usedTotalRunes := 0
	for _, cand := range candidates {
		if selectedTotal >= MultiFileTotalLimit {
			break
		}

		if perFileCounts[cand.FileIndex] >= MultiFilePerFileLimit {
			continue
		}

		chunkRunes := utf8.RuneCountInString(cand.Chunk.DocumentRAGChunk.Text)
		if perFileRunes[cand.FileIndex]+chunkRunes > perFileBudget {
			continue
		}

		if usedTotalRunes+chunkRunes > totalBudget {
			continue
		}

		perFileCounts[cand.FileIndex]++
		perFileRunes[cand.FileIndex] += chunkRunes
		usedTotalRunes += chunkRunes
		selectedTotal++
		selected[cand.FileIndex] = append(selected[cand.FileIndex], cand.Chunk)
	}

	return selected
}

func BuildContextBlockFromMultiFile(fileName string, scored []domain.ScoredDocumentRAGChunk, perFileBudget int, deepMapSummary string) (prompt.DocumentContextBlock, int) {
	return BuildContextBlock(fileName, scored, perFileBudget, deepMapSummary)
}
