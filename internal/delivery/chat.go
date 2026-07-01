package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	"github.com/magomedcoder/coder-server/internal/service"
	"github.com/magomedcoder/coder-server/pkg/contextbudget"
	pkgdomain "github.com/magomedcoder/coder-server/pkg/domain"
	"github.com/magomedcoder/coder-server/pkg/security"
)

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	requestID := resolveRequestID(r.Context(), req.RequestID)

	if err := validateChatRequest(req); err != nil {
		logReq(requestID, "чат отклонён: %v", err)
		writeBadRequest(w, err.Error())
		return
	}

	if h.cfg.ModerationEnabled() {
		texts := make([]string, 0, len(req.Messages)+1)
		if req.System != nil {
			texts = append(texts, *req.System)
		}

		for _, m := range req.Messages {
			texts = append(texts, m.Content)
		}

		if security.ScanMessages(texts) {
			h.recordChatErr()
			logReq(requestID, "чат отклонён: moderation")
			writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("prompt_injection", "запрос отклонён moderation layer"))
			return
		}
	}

	if !*req.Stream && requestID != "" && h.idempotency != nil {
		if status, body, ok := h.idempotency.Get("chat:" + requestID); ok {
			logReq(requestID, "чат idempotency replay status=%d", status)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Idempotent-Replay", "true")
			w.WriteHeader(status)
			_, _ = w.Write(body)
			return
		}
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		h.recordChatErr()
		return
	}

	estimate := estimateChatTokens(req)
	if !h.checkTokenQuota(estimate) {
		h.recordChatErr()
		logReq(requestID, "чат отклонён: превышена дневная квота токенов estimate=%d", estimate)
		writeJSON(w, http.StatusTooManyRequests, domain.NewErrorResponse("quota_exceeded", "превышен дневной лимит токенов"))
		return
	}

	sessionID := ""
	if h.chatSessions != nil {
		sessionID = h.chatSessions.ResolveSessionID(&req)
		req.Messages = h.chatSessions.Merge(sessionID, req.Messages)
	}

	h.enrichContextFromSearch(r.Context(), &req)
	tokenBudget := h.cfg.EffectiveContextTokenBudget(h.llm.ChatHints().MaxContextTokens)
	req.Messages = trimChatHistory(req.Messages, h.cfg.HistoryMaxMessages())
	req.Messages = trimChatHistoryByTokens(req, tokenBudget, generateMaxTokens(req, h.cfg))

	useMCPTools := h.chatMCPLoop != nil && h.chatMCPLoop.Enabled(r.Context(), req.Session)
	logReq(requestID, "чат поток=%v сообщ=%d сессия=%s проект=%q файл=%q сниппетов=%d mcp=%v", *req.Stream, len(req.Messages), sessionLabel(sessionID), chatWorkspaceLabel(req), chatEditorPath(req), chatSnippetCount(req), useMCPTools)
	systemPrompt := h.chatSystemPrompt(req, useMCPTools)
	messages := mapper.RunnerMessages(systemPrompt, req.Messages, req.Editor, req.Context, tokenBudget, h.cfg.ContextScanSecrets(), h.prefixCache)
	genParams := mapper.GenerateParams(req.Generate, h.cfg.Chat.Generate)
	if req.Session != nil && req.Session.Temperature != nil {
		temp := float32(*req.Session.Temperature)
		genParams.Temperature = &temp
	}

	stopSeq := serviceStopSequences(req)
	timeout := h.cfg.ChatTimeoutSeconds()
	if req.Session != nil && req.Session.TimeoutSeconds != nil && *req.Session.TimeoutSeconds > 0 {
		timeout = int32(*req.Session.TimeoutSeconds)
	}

	var serverIDs []int64
	if req.Session != nil {
		serverIDs = req.Session.MCPServerIDs
	}

	if useMCPTools {
		h.runChatWithMCPTools(r, w, req, requestID, sessionID, messages, stopSeq, timeout, genParams, serverIDs)
		return
	}

	streamMeta, ch, err := h.llm.SendMessageWithMeta(r.Context(), messages, stopSeq, timeout, genParams)
	if err != nil {
		h.recordChatErr()
		logReq(requestID, "чат ошибка runner: %v", err)
		if !*req.Stream {
			h.mapRunnerErrorWithQueue(w, err, "chat", requestID, req)
			return
		}

		h.mapRunnerError(w, err)
		return
	}

	if *req.Stream {
		var session *service.StreamSession
		if reg := h.llm.StreamRegistry(); reg != nil {
			session = reg.Start(requestID)
		}
		h.recordChatOK()
		writeRunnerSSE(r.Context(), w, streamMeta, ch, h.activeStreams, session, h.metrics, h.quota, sessionID, requestID, func(content string) {
			h.persistChatSession(sessionID, req.Messages, content)
		})
		return
	}

	var full strings.Builder
	var reasoning strings.Builder
	var usage *domain.TokenUsage
	for chunk := range ch {
		if chunk.ReasoningContent != "" {
			reasoning.WriteString(chunk.ReasoningContent)
		}
		if chunk.Content != "" {
			full.WriteString(chunk.Content)
		}
		if chunk.Usage != nil {
			usage = mapper.TokenUsage(chunk.Usage)
			h.recordTokenUsage(chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens)
		}
	}

	assistant := full.String()
	h.persistChatSession(sessionID, req.Messages, assistant)
	resp := domain.ChatResponse{
		SessionID: sessionID,
		Message: domain.ChatMessage{
			Role:    "assistant",
			Content: assistant,
		},
		Finish: "stop",
		Usage:  usage,
	}
	if rs := reasoning.String(); rs != "" {
		resp.Reasoning = rs
	}

	h.recordChatOK()
	logReq(requestID, "чат готов символов=%d prompt=%v completion=%v", len(assistant), usagePrompt(usage), usageCompletion(usage))
	if requestID != "" && h.idempotency != nil {
		if body, err := json.Marshal(resp); err == nil {
			h.idempotency.Put("chat:"+requestID, http.StatusOK, body)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) runChatWithMCPTools(
	r *http.Request,
	w http.ResponseWriter,
	req domain.ChatRequest,
	requestID, sessionID string,
	messages []*pkgdomain.Message,
	stopSeq []string,
	timeout int32,
	genParams *pkgdomain.GenerationParams,
	serverIDs []int64,
) {
	if *req.Stream {
		var session *service.StreamSession
		if reg := h.llm.StreamRegistry(); reg != nil {
			session = reg.Start(requestID)
		}

		h.recordChatOK()

		writeChatMCPLoopSSE(r.Context(), w, h.chatMCPLoop, messages, stopSeq, timeout, genParams, serverIDs, h.activeStreams, session, h.metrics, h.quota, func(content string) {
			h.persistChatSession(sessionID, req.Messages, content)
		}, sessionID, requestID)
		return
	}

	content, reasoning, usageRaw, err := h.chatMCPLoop.Run(r.Context(), messages, stopSeq, timeout, genParams, serverIDs, requestID, nil, nil)
	if err != nil {
		h.recordChatErr()
		logReq(requestID, "чат MCP ошибка: %v", err)
		h.mapRunnerErrorWithQueue(w, err, "chat", requestID, req)
		return
	}

	h.persistChatSession(sessionID, req.Messages, content)
	resp := domain.ChatResponse{
		SessionID: sessionID,
		Message: domain.ChatMessage{
			Role:    "assistant",
			Content: content,
		},
		Finish: "stop",
		Usage:  mapper.TokenUsage(usageRaw),
	}

	if reasoning != "" {
		resp.Reasoning = reasoning
	}

	if usageRaw != nil {
		h.recordTokenUsage(usageRaw.PromptTokens, usageRaw.CompletionTokens)
	}

	h.recordChatOK()
	logReq(requestID, "чат MCP готов символов=%d prompt=%v completion=%v", len(content), usagePrompt(mapper.TokenUsage(usageRaw)), usageCompletion(mapper.TokenUsage(usageRaw)))
	if requestID != "" && h.idempotency != nil {
		if body, err := json.Marshal(resp); err == nil {
			h.idempotency.Put("chat:"+requestID, http.StatusOK, body)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	requestID := strings.TrimSpace(r.URL.Query().Get("request_id"))
	if requestID == "" {
		writeBadRequest(w, "query request_id обязателен")
		return
	}

	reg := h.llm.StreamRegistry()
	if reg == nil {
		writeJSON(w, http.StatusNotFound, domain.NewErrorResponse("not_found", "поток не найден"))
		return
	}

	session, ok := reg.Get(requestID)
	if !ok {
		writeJSON(w, http.StatusNotFound, domain.NewErrorResponse("not_found", "поток не найден"))
		return
	}

	lastEventID := parseLastEventID(r)
	logReq(requestID, "SSE resume last_event_id=%d done=%v", lastEventID, session.IsDone())
	writeReplaySSE(r.Context(), w, session, lastEventID)
}

func usagePrompt(u *domain.TokenUsage) any {
	if u == nil {
		return "-"
	}

	return u.PromptTokens
}

func usageCompletion(u *domain.TokenUsage) any {
	if u == nil {
		return "-"
	}

	return u.CompletionTokens
}

func validateChatRequest(req domain.ChatRequest) error {
	if req.Stream == nil {
		return errValidation("поле stream обязательно")
	}

	if req.System == nil {
		return errValidation("поле system обязательно")
	}

	if len(req.Messages) == 0 {
		return errValidation("массив messages не должен быть пустым")
	}

	if req.Messages[len(req.Messages)-1].Role != "user" {
		return errValidation(`последнее сообщение в messages должно иметь role "user"`)
	}

	return nil
}

type validationError string

func (e validationError) Error() string { return string(e) }

func errValidation(msg string) error { return validationError(msg) }

func trimChatHistory(messages []domain.ChatMessage, max int) []domain.ChatMessage {
	if max <= 0 || len(messages) <= max {
		return messages
	}

	return messages[len(messages)-max:]
}

func generateMaxTokens(req domain.ChatRequest, cfg *config.Config) int {
	if req.Generate != nil && req.Generate.MaxTokens != nil && *req.Generate.MaxTokens > 0 {
		return *req.Generate.MaxTokens
	}

	if cfg != nil && cfg.Chat.Generate.MaxTokens > 0 {
		return cfg.Chat.Generate.MaxTokens
	}

	return 0
}

func trimChatHistoryByTokens(req domain.ChatRequest, tokenBudget, generateMax int) []domain.ChatMessage {
	if len(req.Messages) == 0 {
		return req.Messages
	}

	systemTokens := 0
	if req.System != nil {
		systemTokens = contextbudget.EstimateTokens(strings.TrimSpace(*req.System))
	}

	contextTokens := estimateContextTokens(req.Editor, req.Context)
	historyBudget := contextbudget.HistoryTokenBudget(tokenBudget, systemTokens, contextTokens, generateMax)

	lines := make([]contextbudget.ChatLine, len(req.Messages))
	for i, m := range req.Messages {
		lines[i] = contextbudget.ChatLine{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	trimmed, changed := contextbudget.TrimChatLines(lines, historyBudget)
	if !changed {
		return req.Messages
	}

	out := make([]domain.ChatMessage, len(trimmed))
	for i, line := range trimmed {
		out[i] = domain.ChatMessage{
			Role:    line.Role,
			Content: line.Content,
		}
	}

	return out
}

func estimateContextTokens(editor *domain.EditorContext, ctx *domain.ChatContext) int {
	total := 0
	if editor != nil {
		total += contextbudget.EstimateTokens(editor.Path)
		total += contextbudget.EstimateTokens(editor.Language)
		total += contextbudget.EstimateTokens(editor.Snippet)
	}

	if ctx == nil {
		return total
	}

	if sel := ctx.Selection; sel != nil {
		total += contextbudget.EstimateTokens(sel.Text)
		total += contextbudget.EstimateTokens(sel.Path)
	}

	for _, sn := range ctx.Snippets {
		total += contextbudget.EstimateTokens(sn.Content)
		total += contextbudget.EstimateTokens(sn.Path)
	}

	if ws := ctx.Workspace; ws != nil {
		total += contextbudget.EstimateTokens(ws.Name)
		total += contextbudget.EstimateTokens(ws.Root)
		total += contextbudget.EstimateTokens(ws.Branch)
	}

	return total
}

func serviceStopSequences(req domain.ChatRequest) []string {
	if req.Session == nil {
		return nil
	}

	return req.Session.StopSequences
}

func estimateChatTokens(req domain.ChatRequest) int64 {
	var n int
	if req.System != nil {
		n += len(*req.System)
	}

	for _, m := range req.Messages {
		n += len(m.Content)
	}

	if req.Editor != nil {
		n += len(req.Editor.Snippet)
	}

	est := int64(n / 4)
	if est < 256 {
		return 256
	}

	return est
}

func (h *Handler) persistChatSession(sessionID string, prior []domain.ChatMessage, assistant string) {
	if h.chatSessions == nil || sessionID == "" {
		return
	}

	out := append([]domain.ChatMessage(nil), prior...)
	if strings.TrimSpace(assistant) != "" {
		out = append(out, domain.ChatMessage{
			Role:    "assistant",
			Content: assistant,
		})
	}

	h.chatSessions.Record(sessionID, out)
}
