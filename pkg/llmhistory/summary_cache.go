package llmhistory

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

const DroppedHistorySummarizerSystem = `Тебе дан фрагмент переписки чата (роли user, assistant, tool). Сожми его в связный пересказ: факты, договорённости, открытые вопросы - чтобы модель могла продолжить без полного текста.
Не более 8 предложений. Без приветствий и метакомментариев. Язык ответа - как у фрагмента.`

func SummaryCacheKey(model, dialogueBody string) string {
	h := sha256.Sum256([]byte(model + "\n\x00\n" + dialogueBody))
	return hex.EncodeToString(h[:])
}

type SummaryCache struct {
	mu      sync.Mutex
	max     int
	keys    []string
	entries map[string]string
}

func NewSummaryCache(maxEntries int) *SummaryCache {
	if maxEntries <= 0 {
		return nil
	}

	return &SummaryCache{
		max:     maxEntries,
		entries: make(map[string]string, maxEntries),
	}
}

func NormalizeSummaryCacheMax(n int) int {
	if n < 0 {
		return 0
	}

	if n > 50_000 {
		return 50_000
	}

	return n
}

func (h *SummaryCache) Get(key string) (string, bool) {
	if h == nil {
		return "", false
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	v, ok := h.entries[key]

	return v, ok
}

func (h *SummaryCache) EnsureMax(m int) {
	if h == nil || m <= 0 {
		return
	}

	m = NormalizeSummaryCacheMax(m)
	if m <= 0 {
		return
	}

	h.mu.Lock()
	if m > h.max {
		h.max = m
	}

	h.mu.Unlock()
}

func (h *SummaryCache) Put(key, summary string) {
	if h == nil || key == "" || summary == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.entries[key]; exists {
		h.entries[key] = summary
		return
	}

	for len(h.keys) >= h.max {
		old := h.keys[0]
		h.keys = h.keys[1:]
		delete(h.entries, old)
	}

	h.entries[key] = summary
	h.keys = append(h.keys, key)
}
