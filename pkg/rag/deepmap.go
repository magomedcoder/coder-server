package rag

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/magomedcoder/gen/pkg/domain"
)

const DeepMapSystem = `Ты сжимаешь выдержки из документа в короткую рабочую заметку для финального шага ответа.

Правила:
- По возможности тот же язык, что и у вопроса пользователя.
- Маркированный список или очень короткие абзацы; без вступлений вроде «Вот».
- Только факты и утверждения, подкреплённые выдержками; при необходимости кратко отметь неуверенность.
- Если выдержки почти не помогают вопросу, ответь одной короткой строкой вроде «Мало релевантного в этом блоке.» (язык подстрой под вопрос).`

const DeepMapMaxExcerptRunesPerChunk = 2800

func FormatChunksForDeepMap(batch []domain.ScoredDocumentRAGChunk) string {
	var b strings.Builder
	for _, sc := range batch {
		body := sc.DocumentRAGChunk.Text
		if utf8.RuneCountInString(body) > DeepMapMaxExcerptRunesPerChunk {
			body = TruncateRunes(body, DeepMapMaxExcerptRunesPerChunk)
		}

		head := sc.DocumentRAGChunk.ChunkIndex
		score := sc.Score
		if score <= NeighborOnlyChunkScore/10 {
			fmt.Fprintf(&b, "--- chunk_index=%d (соседний контекст) ---\n%s\n\n", head, body)
		} else {
			fmt.Fprintf(&b, "--- chunk_index=%d близость=%.4f ---\n%s\n\n", head, score, body)
		}
	}
	return b.String()
}
