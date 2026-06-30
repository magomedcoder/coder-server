package mcpclient

import (
	"strings"
	"testing"
)

func TestRedactMCPLogMessagePayloadBearer(t *testing.T) {
	in := `Authorization: Bearer 1234567890abcdef`
	got := redactMCPLogMessagePayload(in)
	if strings.Contains(got, "eyJ") {
		t.Fatalf("утечка токена: %q", got)
	}

	if !strings.Contains(got, "bearer [REDACTED]") {
		t.Fatalf("ожидалось маркер редактирования: %q", got)
	}
}

func TestRedactMCPLogMessagePayloadCaseInsensitive(t *testing.T) {
	in := "BEARER AAAAAAAAAAAAAAAA"
	got := redactMCPLogMessagePayload(in)
	if strings.Contains(strings.ToUpper(got), "AAAAAAAA") {
		t.Fatalf("получено %q", got)
	}
}

func TestFormatLoggingMessageDataRedactsJSON(t *testing.T) {
	got := formatLoggingMessageData(map[string]any{
		"msg": "token Bearer abcdefghijklmnop",
	})

	if strings.Contains(got, "abcdefghijklmnop") {
		t.Fatalf("получено %q", got)
	}

	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("получено %q", got)
	}
}
