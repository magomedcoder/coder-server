package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/magomedcoder/tce-server/internal/domain"
	"github.com/magomedcoder/tce-server/internal/mapper"
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

	if req.Stream == nil {
		writeBadRequest(w, "поле stream обязательно")
		return
	}

	if req.System == nil {
		writeBadRequest(w, "поле system обязательно")
		return
	}

	if len(req.Messages) == 0 {
		writeBadRequest(w, "массив messages не должен быть пустым")
		return
	}

	if req.Messages[len(req.Messages)-1].Role != "user" {
		writeBadRequest(w, `последнее сообщение в messages должно иметь role "user"`)
		return
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	messages := mapper.RunnerMessages(*req.System, req.Messages, req.Editor)
	genParams := mapper.GenerateParams(req.Generate, h.cfg.Chat.Generate)
	ch, err := h.llm.SendMessage(r.Context(), messages, nil, h.cfg.ChatTimeoutSeconds(), genParams)
	if err != nil {
		mapRunnerError(w, err)
		return
	}

	if *req.Stream {
		writeRunnerSSE(w, ch)
		return
	}

	var full strings.Builder
	for chunk := range ch {
		if chunk.Content != "" {
			full.WriteString(chunk.Content)
		}
	}

	writeJSON(w, http.StatusOK, domain.ChatResponse{
		Message: domain.ChatMessage{
			Role:    "assistant",
			Content: full.String(),
		},
		Finish: "stop",
	})
}
