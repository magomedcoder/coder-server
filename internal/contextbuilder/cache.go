package contextbuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type PrefixCache struct {
	mu      sync.RWMutex
	entries map[string]string
	order   []string
	max     int
}

func NewPrefixCache(maxEntries int) *PrefixCache {
	if maxEntries <= 0 {
		maxEntries = 256
	}

	return &PrefixCache{
		entries: make(map[string]string),
		max:     maxEntries,
	}
}

func (c *PrefixCache) Get(key string) (string, bool) {
	if c == nil || key == "" {
		return "", false
	}

	c.mu.RLock()
	v, ok := c.entries[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *PrefixCache) Put(key, value string) {
	if c == nil || key == "" || value == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[key]; !ok {
		c.order = append(c.order, key)
	}

	c.entries[key] = value
	for len(c.order) > c.max {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
}

func PrefixCacheKey(system string, editor *domain.EditorContext, ctx *domain.ChatContext, tokenBudget int, scanSecrets bool) string {
	payload := struct {
		System      string                `json:"system"`
		Editor      *domain.EditorContext `json:"editor,omitempty"`
		Context     *domain.ChatContext   `json:"context,omitempty"`
		TokenBudget int                   `json:"token_budget"`
		ScanSecrets bool                  `json:"scan_secrets"`
	}{
		System:      system,
		Editor:      editor,
		Context:     ctx,
		TokenBudget: tokenBudget,
		ScanSecrets: scanSecrets,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
