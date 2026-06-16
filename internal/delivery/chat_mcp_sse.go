package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/service"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
)

func writeChatMCPLoopSSE(
	ctx context.Context,
	w http.ResponseWriter,
	loop *service.ChatMCPLoop,
	messages []*gendomain.Message,
	stopSequences []string,
	timeout int32,
	genParams *gendomain.GenerationParams,
	serverIDs []int64,
	streams *ActiveStreams,
	session *service.StreamSession,
	metrics *service.Metrics,
	quota *service.TokenQuota,
	onComplete func(content string),
	sessionID string,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "сервер не поддерживает потоковую передачу",
		})
		return
	}

	if streams != nil {
		streams.Inc()
		defer streams.Dec()
	}

	eventID := 0
	emit := func(event, data string) {
		eventID++
		ev := service.SSEEvent{ID: eventID, Event: event, Data: data}
		if session != nil {
			session.Append(ev)
		}
		fmt.Fprintf(w, "id: %d\n", ev.ID)
		fmt.Fprintf(w, "event: %s\n", event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	onEvent := func(ev service.ChatToolEvent) {
		raw, _ := json.Marshal(ev)
		emit("tool", string(raw))
	}

	onChunk := func(chunk gendomain.LLMStreamChunk) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if chunk.ReasoningContent != "" {
			delta, _ := json.Marshal(map[string]string{"text": chunk.ReasoningContent})
			emit("reasoning", string(delta))
		}

		if chunk.Content != "" {
			delta, _ := json.Marshal(map[string]string{"text": chunk.Content})
			emit("delta", string(delta))
		}

		return true
	}

	content, _, usage, err := loop.Run(ctx, messages, stopSequences, timeout, genParams, serverIDs, onEvent, onChunk)
	if err != nil {
		errData, _ := json.Marshal(map[string]string{"message": err.Error()})
		emit("error", string(errData))
		if session != nil {
			session.MarkDone()
		}

		return
	}

	endData := map[string]any{"finish": "stop"}
	if sessionID != "" {
		endData["session_id"] = sessionID
	}
	if usage != nil {
		endData["usage"] = map[string]int32{
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
		}

		if metrics != nil {
			metrics.RecordTokens(usage.PromptTokens, usage.CompletionTokens)
		}

		if quota != nil {
			quota.Record(int64(usage.PromptTokens) + int64(usage.CompletionTokens))
		}
	}
	_ = content
	if onComplete != nil && strings.TrimSpace(content) != "" {
		onComplete(content)
	}
	
	raw, _ := json.Marshal(endData)
	emit("end", string(raw))
	if session != nil {
		session.MarkDone()
	}
}
