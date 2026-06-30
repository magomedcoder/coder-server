package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/magomedcoder/lmpkg/chatstream"
	"github.com/magomedcoder/lmpkg/domain"
	"github.com/magomedcoder/lmpkg/llmhistory"
	"github.com/magomedcoder/lmpkg/logger"
	"github.com/magomedcoder/lmpkg/mcpclient"
	"github.com/magomedcoder/lmpkg/toolloop"
)

func Run(ctx context.Context, cfg Config) (chan chatstream.ChatStreamChunk, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	maxRounds := cfg.MaxRounds
	if maxRounds <= 0 {
		maxRounds = toolloop.DefaultToolLoopRounds
	}

	if maxRounds > toolloop.MaxToolLoopRoundsCap {
		maxRounds = toolloop.MaxToolLoopRoundsCap
	}

	out := make(chan chatstream.ChatStreamChunk, 64)
	go runLoop(ctx, cfg, maxRounds, out)

	return out, nil
}

func (cfg Config) validate() error {
	if cfg.LLM == nil {
		return fmt.Errorf("agent: LLM repository is nil")
	}

	if cfg.Executor == nil {
		return fmt.Errorf("agent: ToolExecutor is nil")
	}

	if cfg.Store == nil {
		return fmt.Errorf("agent: MessageStore is nil")
	}

	if cfg.GenParams == nil || len(cfg.GenParams.Tools) == 0 {
		return fmt.Errorf("agent: GenParams.Tools пуст")
	}

	if strings.TrimSpace(cfg.RunnerAddr) == "" {
		return fmt.Errorf("agent: RunnerAddr пуст")
	}

	return nil
}

func runLoop(ctx context.Context, cfg Config, maxRounds int, out chan<- chatstream.ChatStreamChunk) {
	defer close(out)

	send := func(chunk chatstream.ChatStreamChunk) bool {
		select {
		case <-ctx.Done():
			return false
		case out <- chunk:
			return true
		}
	}

	sendErr := func(err error) {
		if err == nil {
			return
		}

		logger.W("agent: session_id=%d err=%v", cfg.SessionID, err)
		s := err.Error()
		if s == "" {
			s = "error"
		}

		_ = send(chatstream.ChatStreamChunk{
			Kind: chatstream.StreamChunkKindText,
			Text: s,
		})
	}

	defer func() {
		if r := recover(); r != nil {
			logger.E("agent panic: session_id=%d panic=%v", cfg.SessionID, r)
			sendErr(fmt.Errorf("внутренняя ошибка обработки инструментов"))
		}
	}()

	for _, ch := range cfg.Preface {
		if !send(ch) {
			return
		}
	}

	gp := toolloop.CloneGenParamsForToolCalls(cfg.GenParams)
	history := append([]*domain.Message(nil), cfg.InitialHistory...)
	userTurnVision := mcpclient.LastUserMessageHasVisionAttachment(cfg.InitialHistory)

	logger.I("agent: session_id=%d phase=enter runner=%q history_msgs=%d tools=%d max_rounds=%d", cfg.SessionID, cfg.RunnerAddr, len(history), len(gp.Tools), maxRounds)

	for round := range maxRounds {
		logger.I("agent: session_id=%d round=%d/%d phase=llm_request", cfg.SessionID, round+1, maxRounds)
		ch, err := cfg.LLM.SendMessageOnRunner(ctx, cfg.RunnerAddr, history, cfg.StopSequences, cfg.TimeoutSeconds, toolloop.RunnerInferenceParams(gp, history))
		if err != nil {
			sendErr(err)
			return
		}

		raw, streamed := toolloop.DrainLLMStreamChannelForward(ch, forwardToClient(send))
		full := strings.TrimSpace(raw)
		if full == "" {
			sendErr(fmt.Errorf("модель вернула empty response (tool loop)"))
			return
		}

		blob := toolloop.ExtractToolActionBlob(full)
		logger.I("agent: session_id=%d round=%d assistant_runes=%d blob_bytes=%d", cfg.SessionID, round+1, utf8.RuneCountInString(full), len(blob))

		if blob == "" {
			if err := persistAndComplete(ctx, cfg.Store, cfg.SessionID, full, send, streamed); err != nil {
				sendErr(err)
			}
			return
		}

		done, cont, err := runToolRound(ctx, cfg, gp, &history, full, blob, streamed, userTurnVision, round+1, send, sendErr)
		if err != nil {
			return
		}

		if done {
			return
		}

		if cont {
			continue
		}

		return
	}

	sendErr(fmt.Errorf("превышено число итераций вызова инструментов (%d)", maxRounds))
}

func forwardToClient(send func(chatstream.ChatStreamChunk) bool) func(domain.LLMStreamChunk) bool {
	return func(c domain.LLMStreamChunk) bool {
		if c.ReasoningContent != "" {
			if !send(chatstream.ChatStreamChunk{
				Kind: chatstream.StreamChunkKindReasoning,
				Text: c.ReasoningContent,
			}) {
				return false
			}
		}
		if c.Content != "" {
			return send(chatstream.ChatStreamChunk{
				Kind: chatstream.StreamChunkKindText,
				Text: c.Content,
			})
		}
		return true
	}
}

func persistAndComplete(ctx context.Context, store MessageStore, sessionID int64, content string, send func(chatstream.ChatStreamChunk) bool, streamed bool) error {
	am := domain.NewMessage(sessionID, content, domain.MessageRoleAssistant)
	if err := store.Create(ctx, am); err != nil {
		return err
	}

	toolloop.StreamToolRoundComplete(send, am.Id, streamed, content, content)

	return nil
}

func runToolRound(
	ctx context.Context,
	cfg Config,
	gp *domain.GenerationParams,
	history *[]*domain.Message,
	full, blob string,
	streamed bool,
	userTurnVision bool,
	round int,
	send func(chatstream.ChatStreamChunk) bool,
	sendErr func(error),
) (done, cont bool, err error) {
	hist := *history
	rows, parseErr := toolloop.ParseCohereActionList(blob)
	if parseErr != nil {
		err = persistAndComplete(ctx, cfg.Store, cfg.SessionID, full, send, streamed)
		return err == nil, false, err
	}

	if len(rows) == 0 {
		err = persistAndComplete(ctx, cfg.Store, cfg.SessionID, full, send, streamed)
		return err == nil, false, err
	}

	if len(rows) == 1 && toolloop.IsDirectAnswerTool(rows[0].ToolName) {
		ans := toolloop.DirectAnswerText(rows[0].Parameters)
		if ans == "" {
			ans = full
		}

		err = persistAndComplete(ctx, cfg.Store, cfg.SessionID, ans, send, streamed)
		return err == nil, false, err
	}

	execRows := toolloop.FilterExecutableToolRows(rows)
	if len(execRows) == 0 {
		err = persistAndComplete(ctx, cfg.Store, cfg.SessionID, full, send, streamed)
		return err == nil, false, err
	}

	execCalls, err := toolloop.ResolveExecutableToolCalls(gp, execRows)
	if err != nil {
		sendErr(err)
		return false, false, err
	}

	toolCallsJSON, err := toolloop.ToolCallsToOpenAIJSON(toolloop.ExecutableCallsToActionRows(execCalls))
	if err != nil {
		sendErr(err)
		return false, false, err
	}

	type toolOutcome struct {
		res string
		err error
	}
	outcomes := make([]toolOutcome, len(execCalls))

	loopCtx, stopLoop := context.WithCancel(ctx)
	defer stopLoop()
	var stopOnce sync.Once
	abortTools := func() { stopOnce.Do(stopLoop) }

	var sendMu sync.Mutex
	sendSafe := func(chunk chatstream.ChatStreamChunk) bool {
		sendMu.Lock()
		defer sendMu.Unlock()
		return send(chunk)
	}

	var wg sync.WaitGroup
	for i, call := range execCalls {
		wg.Add(1)
		go func(i int, call toolloop.ExecutableToolCall) {
			defer wg.Done()
			if err := loopCtx.Err(); err != nil {
				outcomes[i].err = err
				return
			}

			st := call.ResolvedName
			if cfg.ToolDisplayName != nil {
				st = cfg.ToolDisplayName(loopCtx, call.ResolvedName, call.RequestedName)
			}

			if !sendSafe(chatstream.ChatStreamChunk{
				Kind:     chatstream.StreamChunkKindToolStatus,
				Text:     "Выполняется: " + st,
				ToolName: st,
			}) {
				abortTools()
				outcomes[i].err = errors.New("отправка статуса клиенту прервана")
				return
			}

			toolCtx, cancelTool := context.WithTimeout(loopCtx, toolloop.ToolExecutionDuration(cfg.TimeoutSeconds))
			defer cancelTool()
			env := &toolloop.LoopEnv{
				RunnerAddr:             cfg.RunnerAddr,
				ResolvedModel:          cfg.SelectedModel,
				SamplingModel:          cfg.SamplingModel,
				StopSequences:          cfg.StopSequences,
				TimeoutSeconds:         cfg.TimeoutSeconds,
				SamplingGen:            toolloop.SamplingGenParamsForMCP(gp),
				UserTurnHasVisionImage: userTurnVision,
			}
			toolCtx = toolloop.WithLoopEnv(toolCtx, env)
			res, execErr := cfg.Executor.Execute(toolCtx, call, gp, env)
			outcomes[i].res = res
			outcomes[i].err = execErr
		}(i, call)
	}
	wg.Wait()

	toolResults := make([]string, len(execCalls))
	failCount := 0
	for i, call := range execCalls {
		if outcomes[i].err != nil {
			failCount++
			deadline := errors.Is(outcomes[i].err, context.DeadlineExceeded)
			toolResults[i] = toolloop.ErrorToolMessage(call, outcomes[i].err, outcomes[i].res, deadline)
			continue
		}
		toolResults[i] = toolloop.TruncateToolResult(outcomes[i].res)
	}

	if failCount > 0 {
		_ = send(chatstream.ChatStreamChunk{
			Kind: chatstream.StreamChunkKindNotice,
			Text: "Один или несколько инструментов завершились с ошибкой. Формирую понятный response...",
		})
	}

	assist := domain.NewMessage(cfg.SessionID, full, domain.MessageRoleAssistant)
	assist.ToolCallsJSON = toolCallsJSON
	toolMsgs := make([]*domain.Message, len(execCalls))
	for i, call := range execCalls {
		tm := domain.NewMessage(cfg.SessionID, toolResults[i], domain.MessageRoleTool)
		tm.ToolName = call.ResolvedName
		tm.ToolCallID = fmt.Sprintf("call_%d", i+1)
		toolMsgs[i] = tm
	}

	if err := cfg.Store.Create(ctx, assist); err != nil {
		sendErr(err)
		return false, false, err
	}

	for _, tm := range toolMsgs {
		if err := cfg.Store.Create(ctx, tm); err != nil {
			sendErr(err)
			return false, false, err
		}
	}

	hist = append(hist, assist)
	hist = append(hist, toolMsgs...)

	if cfg.Cap != nil {
		var trimmed bool
		hist, trimmed, err = cfg.Cap.Cap(ctx, hist, 1+len(toolMsgs))
		if err != nil {
			sendErr(err)
			return false, false, err
		}

		if trimmed {
			_ = send(chatstream.ChatStreamChunk{
				Kind: chatstream.StreamChunkKindNotice,
				Text: llmhistory.HistoryTruncatedClientNotice,
			})
		}
	}

	*history = hist
	logger.I("agent: session_id=%d round=%d phase=round_continue history_msgs=%d", cfg.SessionID, round, len(hist))

	return false, true, nil
}
