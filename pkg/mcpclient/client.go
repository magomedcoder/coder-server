package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/magomedcoder/gen/pkg/mcpclient/domain"
	"net/http"
	"strings"
	"time"

	"github.com/magomedcoder/gen/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DeclaredTool struct {
	Name           string
	Description    string
	ParametersJSON string
}

func parseStringMapJSON(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}

	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}

	if out == nil {
		return map[string]string{}, nil
	}

	return out, nil
}

type headerRoundTripper struct {
	next http.RoundTripper
	h    map[string]string
}

func (t *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.h {
		if strings.TrimSpace(k) != "" && v != "" {
			req.Header.Set(k, v)
		}
	}

	return t.next.RoundTrip(req)
}

func httpClientWithHeaders(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return nil
	}

	base := http.DefaultTransport
	if base == nil {
		base = &http.Transport{}
	}

	return &http.Client{
		Transport: &headerRoundTripper{next: base, h: headers},
		Timeout:   0,
	}
}

func transportFor(ctx context.Context, srv *domain.MCPServer) (mcp.Transport, error) {
	if srv == nil {
		return nil, errors.New("nil mcp server")
	}

	tr := strings.ToLower(strings.TrimSpace(srv.Transport))

	logger.D("MCP transport: server_id=%d name=%q transport=%s timeout_sec=%d", srv.ID, strings.TrimSpace(srv.Name), tr, srv.TimeoutSeconds)

	headers, err := parseStringMapJSON(srv.HeadersJSON)
	if err != nil {
		return nil, fmt.Errorf("headers_json: %w", err)
	}

	hc := httpClientWithHeaders(headers)

	switch tr {
	case "sse":
		raw := strings.TrimSpace(srv.URL)
		if raw == "" {
			return nil, errors.New("sse: пустой url")
		}

		if err := validateHTTPMCPURL("sse", raw); err != nil {
			logger.W("MCP transport sse: server_id=%d url invalid: %v", srv.ID, err)
			return nil, err
		}

		logger.D("MCP transport sse: server_id=%d endpoint_host_ok=true", srv.ID)
		t := &mcp.SSEClientTransport{Endpoint: raw}
		if hc != nil {
			t.HTTPClient = hc
		}

		return t, nil

	case "streamable":
		raw := strings.TrimSpace(srv.URL)
		if raw == "" {
			return nil, errors.New("streamable: пустой url")
		}

		if err := validateHTTPMCPURL("streamable", raw); err != nil {
			logger.W("MCP transport streamable: server_id=%d url invalid: %v", srv.ID, err)
			return nil, err
		}

		logger.D("MCP transport streamable: server_id=%d endpoint_host_ok=true", srv.ID)
		t := &mcp.StreamableClientTransport{
			Endpoint:             raw,
			DisableStandaloneSSE: true,
		}

		if hc != nil {
			t.HTTPClient = hc
		}
		return t, nil

	default:
		logger.W("MCP transport: server_id=%d неизвестный transport %q", srv.ID, tr)
		return nil, fmt.Errorf("неизвестный transport %q", tr)
	}
}

func timeoutFor(srv *domain.MCPServer) time.Duration {
	sec := int64(srv.TimeoutSeconds)
	if sec <= 0 {
		sec = 120
	}

	if sec > 600 {
		sec = 600
	}

	return time.Duration(sec) * time.Second
}

func buildMCPClientOptions(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache) *mcp.ClientOptions {
	caps := &mcp.ClientCapabilities{}
	if r := rootsForSession(); len(r) > 0 {
		caps.RootsV2 = &mcp.RootCapabilities{ListChanged: true}
	}

	opts := &mcp.ClientOptions{
		Capabilities: caps,
	}
	if s := samplingClientOptions(ctx); s != nil {
		opts.CreateMessageHandler = s.CreateMessageHandler
		opts.CreateMessageWithToolsHandler = s.CreateMessageWithToolsHandler
	}

	if n := notifyForListChangedHandlers(notify); n != nil && srv != nil && srv.ID > 0 {
		sid := srv.ID
		opts.ToolListChangedHandler = func(context.Context, *mcp.ToolListChangedRequest) {
			n.InvalidateServerTools(sid)
		}

		opts.ResourceListChangedHandler = func(context.Context, *mcp.ResourceListChangedRequest) {
			n.InvalidateServerResources(sid)
		}

		opts.PromptListChangedHandler = func(context.Context, *mcp.PromptListChangedRequest) {
			n.InvalidateServerPrompts(sid)
		}
	}

	if LogServerMessages() && srv != nil {
		opts.LoggingMessageHandler = loggingMessageHandlerForServer(strings.TrimSpace(srv.Name), srv.ID)
		logger.D("MCP client options: server_id=%d roots=%v sampling=%v log_server_messages=true list_changed_handlers=%v", srv.ID, len(rootsForSession()) > 0, samplingClientOptions(ctx) != nil, notify != nil && srv.ID > 0)
	} else if srv != nil {
		logger.D("MCP client options: server_id=%d roots=%v sampling=%v log_server_messages=false list_changed_handlers=%v", srv.ID, len(rootsForSession()) > 0, samplingClientOptions(ctx) != nil, notify != nil && srv.ID > 0)
	}

	return opts
}

func withEphemeralSession(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache, fn func(context.Context, *mcp.ClientSession) error) error {
	transport, err := transportFor(ctx, srv)
	if err != nil {
		return err
	}

	tctx, cancel := context.WithTimeout(ctx, timeoutFor(srv))
	defer cancel()

	opts := buildMCPClientOptions(ctx, srv, notify)
	cli := mcp.NewClient(&mcp.Implementation{Name: "gen", Version: "1.0.0"}, opts)
	if r := rootsForSession(); len(r) > 0 {
		cli.AddRoots(r...)
	}

	session, err := cli.Connect(tctx, transport, nil)
	if err != nil {
		logger.W("MCP connect (ephemeral): server_id=%d name=%q err=%v", srv.ID, strings.TrimSpace(srv.Name), err)
		return err
	}

	logger.D("MCP connect (ephemeral): server_id=%d name=%q ok", srv.ID, strings.TrimSpace(srv.Name))
	defer func() {
		_ = session.Close()
		logger.D("MCP session close (ephemeral): server_id=%d", srv.ID)
	}()

	return fn(tctx, session)
}

func withSession(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache, fn func(context.Context, *mcp.ClientSession) error) error {
	if useHTTPSessionPool(ctx, srv) {
		logger.D("MCP withSession: server_id=%d режим=http_pool reuse=%v", srv.ID, httpReuseSessions.Load())
		return globalHTTPPool.run(ctx, srv, notify, fn)
	}

	logger.D("MCP withSession: server_id=%d режим=ephemeral", srv.ID)
	return withEphemeralSession(ctx, srv, notify, fn)
}

var callToolSessionRunner = withSession

var callToolInvoker = func(ctx context.Context, session *mcp.ClientSession, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	return session.CallTool(ctx, params)
}

func inputSchemaToParametersJSON(schema any) string {
	if schema == nil {
		return `{"type":"object","properties":{}}`
	}

	b, err := json.Marshal(schema)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return `{"type":"object","properties":{}}`
	}

	return string(b)
}

func ListTools(ctx context.Context, srv *domain.MCPServer) ([]DeclaredTool, error) {
	return listTools(ctx, srv, nil)
}

func listTools(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache) ([]DeclaredTool, error) {
	var out []DeclaredTool
	err := withSession(ctx, srv, notify, func(cctx context.Context, session *mcp.ClientSession) error {
		var cursor string
		for {
			params := &mcp.ListToolsParams{}
			if cursor != "" {
				params.Cursor = cursor
			}

			res, err := session.ListTools(cctx, params)
			if err != nil {
				logger.W("MCP listTools: server_id=%d name=%q err=%v", srv.ID, strings.TrimSpace(srv.Name), err)
				return err
			}

			for _, t := range res.Tools {
				if t == nil || strings.TrimSpace(t.Name) == "" {
					continue
				}

				out = append(out, DeclaredTool{
					Name:           t.Name,
					Description:    strings.TrimSpace(t.Description),
					ParametersJSON: inputSchemaToParametersJSON(t.InputSchema),
				})
			}

			cursor = strings.TrimSpace(res.NextCursor)
			if cursor == "" {
				break
			}
		}

		sid := int64(0)
		snm := ""
		if srv != nil {
			sid = srv.ID
			snm = strings.TrimSpace(srv.Name)
		}

		logger.D("MCP listTools: server_id=%d name=%q объявлено_инструментов=%d", sid, snm, len(out))

		return nil
	})

	recordListTools(err)
	return out, err
}

func CallTool(ctx context.Context, srv *domain.MCPServer, mcpToolName string, arguments json.RawMessage, notify ...*ToolsListCache) (string, error) {
	started := time.Now()
	defer func() {
		recordCallToolDuration(time.Since(started))
	}()

	serverID := int64(0)
	if srv != nil {
		serverID = srv.ID
	}

	if strings.TrimSpace(mcpToolName) == "" {
		recordCallToolTransportErr()
		recordCallToolServer(serverID, "transport_err")
		return "", errors.New("пустое имя инструмента MCP")
	}

	argBytes := len(arguments)

	var nc *ToolsListCache
	if len(notify) > 0 {
		nc = notify[0]
	}

	var result string
	var callErr error
	attempts := 1
	if callToolAllowsTransportRetry(mcpToolName) {
		attempts = 2
	}

	srvName := ""
	if srv != nil {
		srvName = strings.TrimSpace(srv.Name)
	}
	logger.I("MCP CallTool: phase=start server_id=%d server_name=%q tool=%q args_bytes=%d attempts_max=%d http_pool=%t", serverID, srvName, mcpToolName, argBytes, attempts, useHTTPSessionPool(ctx, srv))

	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			logger.I("MCP CallTool: phase=retry server_id=%d tool=%q attempt=%d/%d", serverID, mcpToolName, attempt+1, attempts)
		}
		err = callToolSessionRunner(ctx, srv, nc, func(cctx context.Context, session *mcp.ClientSession) error {
			var args any
			if len(arguments) > 0 {
				if err := json.Unmarshal(arguments, &args); err != nil {
					return fmt.Errorf("аргументы инструмента: %w", err)
				}
			}

			if args == nil {
				args = map[string]any{}
			}

			logger.I("MCP CallTool: phase=invoke server_id=%d tool=%q", serverID, mcpToolName)
			res, err := callToolInvoker(cctx, session, &mcp.CallToolParams{
				Name:      mcpToolName,
				Arguments: args,
			})
			if err != nil {
				return err
			}

			result = CallToolResultString(res)
			if res != nil && res.IsError {
				callErr = errors.New(strings.TrimSpace(result))
			}
			isErr := res != nil && res.IsError
			logger.I("MCP CallTool: phase=response server_id=%d tool=%q is_mcp_error=%v reply_bytes=%d", serverID, mcpToolName, isErr, len(result))

			return nil
		})
		if err == nil {
			break
		}
		if attempt < attempts-1 && isRetryableTransportError(ctx, err) {
			logger.W("MCP CallTool: server_id=%d tool=%q transport_retry attempt=%d/%d err=%v", serverID, mcpToolName, attempt+1, attempts, err)
			recordCallToolRetry()
			continue
		}
	}

	if err != nil {
		logger.W("MCP CallTool: phase=finish server_id=%d tool=%q outcome=transport_err duration=%s err=%v", serverID, mcpToolName, time.Since(started), err)
		recordCallToolTransportErr()
		recordCallToolServer(serverID, "transport_err")
		return "", err
	}

	if callErr != nil {
		logger.W("MCP CallTool: phase=finish server_id=%d tool=%q outcome=mcp_error duration=%s reply_bytes=%d err=%v", serverID, mcpToolName, time.Since(started), len(result), callErr)
		recordCallToolMCPError()
		recordCallToolServer(serverID, "mcp_error")
		return result, callErr
	}

	logger.I("MCP CallTool: phase=finish server_id=%d tool=%q outcome=ok duration=%s reply_bytes=%d", serverID, mcpToolName, time.Since(started), len(result))
	recordCallToolOK()
	recordCallToolServer(serverID, "ok")
	return result, nil
}
