package mcpclient

import (
	"github.com/magomedcoder/lmpkg/mcpclient/domain"
	"testing"
	"time"
)

func TestServerConfigFingerprintChangesWithCommand(t *testing.T) {
	a := &domain.MCPServer{
		Transport:      "streamable",
		URL:            "http://127.0.0.1:9/x",
		HeadersJSON:    `{}`,
		TimeoutSeconds: 120,
	}

	fp1 := serverConfigFingerprint(a)
	a.URL = "http://127.0.0.1:9/y"
	fp2 := serverConfigFingerprint(a)
	if fp1 == fp2 {
		t.Fatal("fingerprint должно меняться when url changes")
	}
}

func TestToolsListCacheInvalidateServerID(t *testing.T) {
	c := NewToolsListCache()
	key := listCacheKey{
		id: 1,
		fp: "abc",
	}

	c.mu.Lock()
	c.toolEntries[key] = toolsCacheEntry{until: time.Now().Add(time.Hour)}
	c.resEntries[key] = resourcesCacheEntry{until: time.Now().Add(time.Hour)}
	c.promptsEntries[key] = promptsCacheEntry{until: time.Now().Add(time.Hour)}
	c.mu.Unlock()
	c.InvalidateServerID(1)
	c.mu.RLock()
	_, okTools := c.toolEntries[key]
	_, okRes := c.resEntries[key]
	_, okPr := c.promptsEntries[key]
	c.mu.RUnlock()

	if okTools || okRes || okPr {
		t.Fatal("ожидается удаление ключей tools/resources/prompts")
	}
}

func TestNotifyForListChangedHandlers(t *testing.T) {
	t.Cleanup(func() { SetToolsListCacheForNotifications(nil) })

	SetToolsListCacheForNotifications(nil)
	if g := notifyForListChangedHandlers(nil); g != nil {
		t.Fatalf("ожидалось nil, получено %p", g)
	}

	c := NewToolsListCache()
	SetToolsListCacheForNotifications(c)
	if g := notifyForListChangedHandlers(nil); g != c {
		t.Fatal("ожидалось кэш process по умолчанию")
	}
	if g := notifyForListChangedHandlers(c); g != c {
		t.Fatal("explicit notify должно перекрывать по умолчанию")
	}
}
