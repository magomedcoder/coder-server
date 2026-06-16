package domain

import (
	"encoding/json"
	"testing"
)

var documentedErrorCodes = []string{
	"bad_request",
	"unauthorized",
	"not_found",
	"method_not_allowed",
	"rate_limit_exceeded",
	"quota_exceeded",
	"prompt_injection",
	"mcp_error",
	"forbidden",
	"overloaded",
	"internal_error",
	"service_unavailable",
	"invalid_request",
	"timeout",
}

func TestDocumentedErrorCodesShape(t *testing.T) {
	for _, code := range documentedErrorCodes {
		body, err := json.Marshal(NewErrorResponse(code, "test message"))
		if err != nil {
			t.Fatalf("marshal %q: %v", code, err)
		}

		var parsed struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("unmarshal %q: %v", code, err)
		}
		
		if parsed.Error.Code != code {
			t.Fatalf("code %q: got %q", code, parsed.Error.Code)
		}

		if parsed.Error.Message != "test message" {
			t.Fatalf("message for %q: got %q", code, parsed.Error.Message)
		}
	}
}
