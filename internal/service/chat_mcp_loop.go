package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/pkg/mcpregistry"
	gendomain "github.com/magomedcoder/lmpkg/domain"
	"github.com/magomedcoder/lmpkg/toolloop"
)

const defaultChatToolRounds = 5

type ChatToolEvent struct {
	Phase string `json:"phase"`
	Tool  string `json:"tool,omitempty"`
	OK    bool   `json:"ok,omitempty"`
	Text  string `json:"text,omitempty"`
}

type ChatMCPLoop struct {
	llm       *LLMRunnerService
	mcp       *MCPRegistry
	maxRounds int
}

func NewChatMCPLoop(llm *LLMRunnerService, mcp *MCPRegistry, maxRounds int) *ChatMCPLoop {
	if maxRounds <= 0 {
		maxRounds = defaultChatToolRounds
	}

	return &ChatMCPLoop{
		llm:       llm,
		mcp:       mcp,
		maxRounds: maxRounds,
	}
}

func (l *ChatMCPLoop) Enabled(ctx context.Context, session *domain.ChatSession) bool {
	if l == nil || l.llm == nil || l.mcp == nil || !l.mcp.Enabled() {
		return false
	}

	if session == nil || session.MCPEnabled == nil || !*session.MCPEnabled {
		return false
	}

	tools, err := l.mcp.GenerationTools(ctx, session.MCPServerIDs)
	return err == nil && len(tools) > 0
}

func (l *ChatMCPLoop) Run(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
	serverIDs []int64,
	requestID string,
	onEvent func(ChatToolEvent),
	onChunk func(gendomain.LLMStreamChunk) bool,
) (content, reasoning string, usage *gendomain.StreamTokenUsage, err error) {
	if l == nil || l.llm == nil || l.mcp == nil {
		return "", "", nil, fmt.Errorf("MCP tool loop не инициализирован")
	}

	tools, err := l.mcp.GenerationTools(ctx, serverIDs)
	if err != nil {
		return "", "", nil, err
	}

	if len(tools) == 0 {
		return "", "", nil, fmt.Errorf("нет MCP-инструментов для вызова")
	}

	_, runnerAddr, err := l.llm.ProbeBestRunner(ctx)
	if err != nil {
		return "", "", nil, err
	}

	gp := toolloop.CloneGenParamsForToolCalls(genParams)
	gp.Tools = tools

	history := append([]*gendomain.Message(nil), messages...)
	executor := &mcpChatExecutor{mcp: l.mcp}

	emitTool := func(ev ChatToolEvent) {
		if requestID != "" {
			log.Printf("request_id=%s mcp_tool этап=%s инструмент=%s успех=%v", requestID, ev.Phase, ev.Tool, ev.OK)
		}

		if onEvent != nil {
			onEvent(ev)
		}
	}

	for round := 0; round < l.maxRounds; round++ {
		if err := ctx.Err(); err != nil {
			return "", reasoning, usage, err
		}

		ch, sendErr := l.llm.SendMessageOnRunner(ctx, runnerAddr, history, stopSequences, timeoutSeconds, toolloop.RunnerInferenceParams(gp, history))
		if sendErr != nil {
			return "", "", usage, sendErr
		}

		forward := func(c gendomain.LLMStreamChunk) bool {
			if c.ReasoningContent != "" {
				reasoning += c.ReasoningContent
			}

			if c.Usage != nil {
				usage = c.Usage
			}

			if onChunk == nil {
				return true
			}

			return onChunk(c)
		}

		raw, _ := toolloop.DrainLLMStreamChannelForward(ch, forward)
		full := strings.TrimSpace(raw)
		if full == "" {
			return "", reasoning, usage, fmt.Errorf("модель вернула пустой ответ")
		}

		blob := toolloop.ExtractToolActionBlob(full)
		if blob == "" {
			content = full
			return content, reasoning, usage, nil
		}

		rows, parseErr := toolloop.ParseCohereActionList(blob)
		if parseErr != nil || len(rows) == 0 {
			content = full
			return content, reasoning, usage, nil
		}

		if len(rows) == 1 && toolloop.IsDirectAnswerTool(rows[0].ToolName) {
			ans := toolloop.DirectAnswerText(rows[0].Parameters)
			if ans == "" {
				ans = full
			}

			content = ans
			return content, reasoning, usage, nil
		}

		execRows := toolloop.FilterExecutableToolRows(rows)
		if len(execRows) == 0 {
			content = full
			return content, reasoning, usage, nil
		}

		execCalls, resolveErr := toolloop.ResolveExecutableToolCalls(gp, execRows)
		if resolveErr != nil {
			return "", reasoning, usage, resolveErr
		}

		toolCallsJSON, jsonErr := toolloop.ToolCallsToOpenAIJSON(toolloop.ExecutableCallsToActionRows(execCalls))
		if jsonErr != nil {
			return "", reasoning, usage, jsonErr
		}

		type toolOutcome struct {
			res string
			err error
		}
		outcomes := make([]toolOutcome, len(execCalls))

		loopCtx, stopLoop := context.WithCancel(ctx)
		var stopOnce sync.Once
		abortTools := func() { stopOnce.Do(stopLoop) }

		var wg sync.WaitGroup
		for i, call := range execCalls {
			wg.Add(1)
			go func(i int, call toolloop.ExecutableToolCall) {
				defer wg.Done()
				if loopCtx.Err() != nil {
					outcomes[i].err = loopCtx.Err()
					return
				}

				emitTool(ChatToolEvent{
					Phase: "start",
					Tool:  call.ResolvedName,
				})
				toolCtx, cancelTool := context.WithTimeout(loopCtx, toolloop.ToolExecutionDuration(timeoutSeconds))
				defer cancelTool()

				res, execErr := executor.Execute(toolCtx, call)
				outcomes[i].res = res
				outcomes[i].err = execErr
				emitTool(ChatToolEvent{
					Phase: "done",
					Tool:  call.ResolvedName,
					OK:    execErr == nil,
					Text:  truncateToolPreview(res),
				})
				if execErr != nil {
					log.Printf("request_id=%s mcp_tool инструмент=%q ошибка=%v", requestID, call.ResolvedName, execErr)
					abortTools()
				}
			}(i, call)
		}
		wg.Wait()
		stopLoop()

		toolResults := make([]string, len(execCalls))
		for i, call := range execCalls {
			if outcomes[i].err != nil {
				timedOut := errors.Is(outcomes[i].err, context.DeadlineExceeded)
				toolResults[i] = toolloop.ErrorToolMessage(call, outcomes[i].err, outcomes[i].res, timedOut)
				continue
			}

			toolResults[i] = toolloop.TruncateToolResult(outcomes[i].res)
		}

		assist := &gendomain.Message{
			Content:       full,
			Role:          gendomain.MessageRoleAssistant,
			ToolCallsJSON: toolCallsJSON,
		}
		toolMsgs := make([]*gendomain.Message, len(execCalls))
		for i, call := range execCalls {
			toolMsgs[i] = &gendomain.Message{
				Content:    toolResults[i],
				Role:       gendomain.MessageRoleTool,
				ToolName:   call.ResolvedName,
				ToolCallID: fmt.Sprintf("call_%d", i+1),
			}
		}

		history = append(history, assist)
		history = append(history, toolMsgs...)
	}

	return "", reasoning, usage, fmt.Errorf("превышено число итераций MCP (%d)", l.maxRounds)
}

func truncateToolPreview(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 120 {
		return s
	}

	return s[:120] + "..."
}

type mcpChatExecutor struct {
	mcp *MCPRegistry
}

func (e *mcpChatExecutor) Execute(ctx context.Context, call toolloop.ExecutableToolCall) (string, error) {
	if e == nil || e.mcp == nil {
		return "", fmt.Errorf("MCP registry не инициализирован")
	}

	if !e.mcp.HasTool(call.ResolvedName) {
		return "", fmt.Errorf("неизвестный инструмент %q", call.ResolvedName)
	}

	var args map[string]any
	if len(call.Parameters) > 0 {
		if err := json.Unmarshal(call.Parameters, &args); err != nil {
			return "", fmt.Errorf("arguments: %w", err)
		}
	}

	if args == nil {
		args = map[string]any{}
	}

	return e.mcp.Call(ctx, mcpregistry.CallRequest{
		Tool:      call.ResolvedName,
		Arguments: args,
	})
}
