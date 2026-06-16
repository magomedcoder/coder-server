package delivery

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	"github.com/magomedcoder/coder-server/internal/service"
	"github.com/magomedcoder/coder-server/pkg/security"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
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
	log.Printf("request_id=%s chat stream=%v messages=%d", requestID, req.Stream, len(req.Messages))

	if err := validateChatRequest(req); err != nil {
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
			writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("prompt_injection", "запрос отклонён moderation layer"))
			return
		}
	}

	if !*req.Stream && requestID != "" && h.idempotency != nil {
		if status, body, ok := h.idempotency.Get("chat:" + requestID); ok {
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
		writeJSON(w, http.StatusTooManyRequests, domain.NewErrorResponse("quota_exceeded", "превышен дневной лимит токенов"))
		return
	}

	sessionID := ""
	if h.chatSessions != nil {
		sessionID = h.chatSessions.ResolveSessionID(&req)
		req.Messages = h.chatSessions.Merge(sessionID, req.Messages)
	}

	h.enrichContextFromSearch(r.Context(), &req)
	req.Messages = trimChatHistory(req.Messages, h.cfg.HistoryMaxMessages())

	useMCPTools := h.chatMCPLoop != nil && h.chatMCPLoop.Enabled(r.Context(), req.Session)
	systemPrompt := h.chatSystemPrompt(req, useMCPTools)
	messages := mapper.RunnerMessages(systemPrompt, req.Messages, req.Editor, req.Context, h.cfg.ContextTokenBudget(), h.cfg.ContextScanSecrets(), h.prefixCache)
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

	ch, err := h.llm.SendMessage(r.Context(), messages, stopSeq, timeout, genParams)
	if err != nil {
		h.recordChatErr()
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
		writeRunnerSSE(r.Context(), w, ch, h.activeStreams, session, h.metrics, h.quota, sessionID, func(content string) {
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
	messages []*gendomain.Message,
	stopSeq []string,
	timeout int32,
	genParams *gendomain.GenerationParams,
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
		}, sessionID)
		return
	}

	content, reasoning, usageRaw, err := h.chatMCPLoop.Run(r.Context(), messages, stopSeq, timeout, genParams, serverIDs, nil, nil)
	if err != nil {
		h.recordChatErr()
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
	writeReplaySSE(r.Context(), w, session, lastEventID)
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
			Role: "assistant",
			Content: assistant,
		})
	}

	h.chatSessions.Record(sessionID, out)
}
