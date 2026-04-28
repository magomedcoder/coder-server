package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
)

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	ok, err := a.llm.CheckConnection(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorBody("internal_error", "gen-runner недоступен"))
		return
	}

	loaded := false
	if ok {
		if err := a.llm.ModelReady(r.Context()); err == nil {
			loaded = true
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": ok && loaded,
	})
}

func (a *App) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req ChatRequest
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

	if !a.ensureRunnerReady(r.Context(), w) {
		return
	}

	messages := makeRunnerMessages(*req.System, req.Messages, req.Editor)
	genParams := mapGenerateParams(req.Generate)
	ch, err := a.llm.SendMessage(r.Context(), messages, nil, 0, genParams)
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

	writeJSON(w, http.StatusOK, ChatResponse{
		Message: ChatMessage{
			Role:    "assistant",
			Content: full.String(),
		},
		Finish: "stop",
	})
}

func (a *App) handleAgentStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req AgentStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if !a.ensureRunnerReady(r.Context(), w) {
		return
	}

	resp, err := a.runAgentStep(r.Context(), req)
	if err != nil {
		mapRunnerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *App) runAgentStep(ctx context.Context, req AgentStepRequest) (AgentStepResponse, error) {
	system := agentStepSystemPrompt()
	user := buildAgentStepUserPrompt(req)

	messages := makeRunnerMessages(system, []ChatMessage{
		{Role: "user", Content: user},
	}, nil)

	genParams := &domain.GenerationParams{}
	maxTokens := int32(2048)
	temp := float32(0.1)
	genParams.MaxTokens = &maxTokens
	genParams.Temperature = &temp
	rfType := "json_object"
	genParams.ResponseFormat = &domain.ResponseFormat{Type: rfType}

	raw, err := a.llm.CollectMessage(ctx, messages, nil, 0, genParams)
	if err != nil {
		return AgentStepResponse{}, err
	}

	parsed, err := parseAgentStepLLMOutput(raw)
	if err != nil {
		return AgentStepResponse{
			Finish:  true,
			Summary: strings.TrimSpace(raw),
			Calls:   nil,
		}, nil
	}

	return parsed, nil
}

const agentStepSystemPromptText = `You are an autonomous coding agent for the Tce editor.
Respond with a single JSON object only (no markdown fences):
{"finish":boolean,"summary":string,"calls":[{"tool":string,"id":string,"args":object}]}
Available tools: list_dir, read_file, glob_search, search_content, apply_patch, create_file, run_command.
If the goal is complete, set finish=true and calls=[].
If more work is needed, set finish=false and list tool calls with unique id fields like "call-1".`

func agentStepSystemPrompt() string {
	return agentStepSystemPromptText
}

func buildAgentStepUserPrompt(req AgentStepRequest) string {
	var b strings.Builder
	if g := strings.TrimSpace(req.Goal); g != "" {
		fmt.Fprintf(&b, "Goal: %s\n", g)
	}

	if sid := strings.TrimSpace(req.SessionID); sid != "" {
		fmt.Fprintf(&b, "Session: %s\n", sid)
	}

	if len(req.Context) > 0 {
		ctxJSON, _ := json.Marshal(req.Context)
		fmt.Fprintf(&b, "Context: %s\n", string(ctxJSON))
	}

	if len(req.Observations) > 0 {
		b.WriteString("Observations from previous tool calls:\n")
		for _, obs := range req.Observations {
			line, _ := json.Marshal(obs)
			b.Write(line)
			b.WriteByte('\n')
		}
	}
	if b.Len() == 0 {
		return "Continue the agent session."
	}

	return b.String()
}

func parseAgentStepLLMOutput(raw string) (AgentStepResponse, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var resp AgentStepResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return AgentStepResponse{}, err
	}

	if resp.Calls == nil {
		resp.Calls = []AgentToolCall{}
	}

	return resp, nil
}

func writeRunnerSSE(w http.ResponseWriter, chunks <-chan domain.LLMStreamChunk) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "сервер не поддерживает потоковую передачу",
		})
		return
	}

	for chunk := range chunks {
		if chunk.Content == "" {
			continue
		}

		fmt.Fprintf(w, "event: delta\n")
		fmt.Fprintf(w, "data: {\"text\":%q}\n\n", chunk.Content)
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: end\n")
	fmt.Fprintf(w, "data: {\"finish\":\"stop\"}\n\n")
	flusher.Flush()
}
