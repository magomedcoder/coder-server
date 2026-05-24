package delivery

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	"github.com/magomedcoder/coder-server/internal/service"
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

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	messages := mapper.RunnerMessages(*req.System, req.Messages, req.Editor, req.Context, h.cfg.ContextTokenBudget())
	genParams := mapper.GenerateParams(req.Generate, h.cfg.Chat.Generate)
	ch, err := h.llm.SendMessage(r.Context(), messages, nil, h.cfg.ChatTimeoutSeconds(), genParams)
	if err != nil {
		h.mapRunnerError(w, err)
		return
	}

	if *req.Stream {
		var session *service.StreamSession
		if reg := h.llm.StreamRegistry(); reg != nil {
			session = reg.Start(requestID)
		}
		writeRunnerSSE(r.Context(), w, ch, h.activeStreams, session)
		return
	}

	var full strings.Builder
	var usage *domain.TokenUsage
	for chunk := range ch {
		if chunk.Content != "" {
			full.WriteString(chunk.Content)
		}
		if chunk.Usage != nil {
			usage = mapper.TokenUsage(chunk.Usage)
		}
	}

	writeJSON(w, http.StatusOK, domain.ChatResponse{
		Message: domain.ChatMessage{
			Role:    "assistant",
			Content: full.String(),
		},
		Finish: "stop",
		Usage:  usage,
	})
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
