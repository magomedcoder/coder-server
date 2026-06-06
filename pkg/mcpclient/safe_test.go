package mcpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRecoverPanic_PassesThroughOK(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	srv := httptest.NewServer(RecoverPanic("test-origin", inner))

	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("статус: получено %d ожидалось %d", resp.StatusCode, http.StatusTeapot)
	}
}

func TestRecoverPanic_SecondRequestAfterPanicStillWorks(t *testing.T) {
	var n int
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			panic("deliberate")
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(RecoverPanic("panic-test", inner))
	t.Cleanup(srv.Close)

	_, err := http.Get(srv.URL)
	if err != nil {
		t.Logf("первый запрос после panic обработчика (допустимо flaky): %v", err)
	}

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("второй запрос: получено статус %d", resp.StatusCode)
	}
}

func TestRecoverPanic_EmptyOriginUsesDefault(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := RecoverPanic("   ", inner)
	if h == nil {
		t.Fatal("ожидалось обработчик не nil")
	}
	srv := httptest.NewServer(h)

	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("статус: %d", resp.StatusCode)
	}
}

func TestSafeToolInvokePanicBecomesToolError(t *testing.T) {
	res, meta, err := SafeToolInvoke("test-srv", "panic_tool", func() (*mcp.CallToolResult, any, error) {
		panic("boom")
	})

	if err != nil {
		t.Fatalf("ожидалось err=nil, получено %v", err)
	}

	if meta != nil {
		t.Fatalf("ожидался meta nil, получено %#v", meta)
	}

	if res == nil || !res.IsError {
		t.Fatalf("ожидалось IsError tool result, получено %#v", res)
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
		t.Fatalf("неожиданное: %#v", res)
	}
}

func TestSafeToolInvokeEmptyOriginUsesDefault(t *testing.T) {
	res, _, err := SafeToolInvoke("", "t", func() (*mcp.CallToolResult, any, error) {
		panic("x")
	})

	if err != nil || res == nil || !res.IsError {
		t.Fatalf("ожидалось tool error, получено res=%#v err=%v", res, err)
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
		t.Fatalf("ожидалось ошибка обработчик, res=%#v err=%v", res, err)
	}
}
