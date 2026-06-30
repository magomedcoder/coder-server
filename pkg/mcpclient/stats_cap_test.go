package mcpclient

import (
	"strings"
	"testing"
)

func resetPerServerCallStatsForTest(t *testing.T) {
	t.Helper()
	callToolByServer.Range(func(k, v any) bool {
		callToolByServer.Delete(k)
		return true
	})
	callToolStatDistinctCount.Store(0)
}

func TestRecordCallToolServerRespectsMaxDistinct(t *testing.T) {
	resetPerServerCallStatsForTest(t)
	defer resetPerServerCallStatsForTest(t)
	SetMaxTrackedCallStatServerIDs(2)
	defer SetMaxTrackedCallStatServerIDs(0)

	recordCallToolServer(10, "ok")
	recordCallToolServer(20, "ok")
	recordCallToolServer(30, "ok")

	m := MCPCountersMap()
	if m["call_tool_server_10_ok"] < 1 || m["call_tool_server_20_ok"] < 1 {
		t.Fatalf("ожидалось статистика для 10 и 20: %v", m)
	}

	if m["call_tool_server_30_ok"] != 0 {
		t.Fatalf("сервер 30 должно быть capped out: %v", m)
	}

	for k := range m {
		if strings.Contains(k, "call_tool_server_30_") {
			t.Fatalf("неожиданный key for capped сервер: %s", k)
		}
	}
}
