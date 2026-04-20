package mcpsafe

import (
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSafeToolInvokePanicBecomesToolError(t *testing.T) {
	res, meta, err := SafeToolInvoke("test-srv", "panic_tool", func() (*mcp.CallToolResult, any, error) {
		panic("boom")
	})

	if err != nil {
		t.Fatalf("expected err=nil, got %v", err)
	}

	if meta != nil {
		t.Fatalf("expected meta nil, got %#v", meta)
	}

	if res == nil || !res.IsError {
		t.Fatalf("expected IsError tool result, got %#v", res)
	}
}

func TestSafeToolInvokeSuccess(t *testing.T) {
	res, meta, err := SafeToolInvoke("test-srv", "ok", func() (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: `{"ok":true}`}},
		}, nil, nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if meta != nil {
		t.Fatalf("meta: %v", meta)
	}

	if res == nil || res.IsError {
		t.Fatalf("unexpected: %#v", res)
	}
}

func TestSafeToolInvokeEmptyOriginUsesDefault(t *testing.T) {
	res, _, err := SafeToolInvoke("", "t", func() (*mcp.CallToolResult, any, error) {
		panic("x")
	})

	if err != nil || res == nil || !res.IsError {
		t.Fatalf("want tool error, got res=%#v err=%v", res, err)
	}
}

func TestSafeToolInvokePassesThroughHandlerError(t *testing.T) {
	e := errors.New("business")
	res, meta, err := SafeToolInvoke("test-srv", "err", func() (*mcp.CallToolResult, any, error) {
		return nil, nil, e
	})

	if meta != nil {
		t.Fatalf("meta: %v", meta)
	}

	if err != e {
		t.Fatalf("expected handler error, got res=%#v err=%v", res, err)
	}
}
