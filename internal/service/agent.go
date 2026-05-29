package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
)

type AgentService struct {
	llm            *LLMRunnerService
	cfg            config.AgentConfig
	timeoutSeconds int32
	tokenBudget    int
	scanSecrets    bool
	sessions       *AgentSessionStore
	policy         *AgentPolicy
	mcp            *MCPRegistry
}

func NewAgentService(llm *LLMRunnerService, cfg *config.Config, mcp *MCPRegistry) *AgentService {
	return &AgentService{
		llm:            llm,
		cfg:            cfg.Agent,
		timeoutSeconds: cfg.ChatTimeoutSeconds(),
		tokenBudget:    cfg.ContextTokenBudget(),
		scanSecrets:    cfg.ContextScanSecrets(),
		sessions:       NewAgentSessionStore(cfg.Agent.MaxSteps),
		policy:         NewAgentPolicy(cfg.Agent.AllowedPaths, cfg.Agent.BlockedCommands, cfg.Agent.AllowedCommands),
		mcp:            mcp,
	}
}

func (s *AgentService) Policy() *AgentPolicy {
	if s == nil {
		return nil
	}

	return s.policy
}

func (s *AgentService) Step(ctx context.Context, req domain.AgentStepRequest) (domain.AgentStepResponse, error) {
	if s == nil {
		return domain.AgentStepResponse{}, fmt.Errorf("agent service не инициализирован")
	}

	sessionID := strings.TrimSpace(req.SessionID)
	step := 0
	if sessionID != "" {
		var limited domain.AgentStepResponse
		var stop bool
		step, limited, stop = s.sessions.BeginStep(sessionID, req.Goal)
		if stop {
			return limited, nil
		}
	}

	system := s.systemPrompt()
	user := s.buildAgentStepUserPrompt(req)

	messages := mapper.RunnerMessages(system, []domain.ChatMessage{
		{Role: "user", Content: user},
	}, nil, nil, s.tokenBudget, s.scanSecrets, nil)

	result, err := s.llm.CollectMessage(ctx, messages, nil, s.timeoutSeconds, s.agentGenerationParams())
	if err != nil {
		return domain.AgentStepResponse{}, err
	}

	parsed, err := parseAgentStepLLMOutput(result.Content)
	if err != nil {
		resp := domain.AgentStepResponse{
			Finish:  true,
			Summary: strings.TrimSpace(result.Content),
			Calls:   nil,
			Step:    step,
		}
		if sessionID != "" {
			s.sessions.Reset(sessionID)
		}
		return resp, nil
	}

	parsed.Calls = s.filterAgentToolCalls(parsed.Calls)
	filtered, blocked := s.policy.FilterCalls(parsed.Calls)
	parsed.Calls = filtered
	parsed.Blocked = blocked
	parsed.Step = step

	if sessionID != "" && strings.TrimSpace(parsed.Summary) != "" {
		s.sessions.SetSummary(sessionID, parsed.Summary)
	}

	if parsed.Finish && sessionID != "" {
		s.sessions.Reset(sessionID)
	}

	return parsed, nil
}

func (s *AgentService) filterAgentToolCalls(calls []domain.AgentToolCall) []domain.AgentToolCall {
	if len(calls) == 0 {
		return []domain.AgentToolCall{}
	}

	out := make([]domain.AgentToolCall, 0, len(calls))
	for _, call := range calls {
		if IsKnownAgentTool(call.Tool, s.mcp) {
			out = append(out, call)
		}
	}
	return out
}

func (s *AgentService) systemPrompt() string {
	base := agentStepSystemPromptText
	if s != nil && s.mcp != nil {
		if block := s.mcp.ToolsPromptBlock(); block != "" {
			base += "\n\n" + block
		}
	}
	return base
}

func (s *AgentService) agentGenerationParams() *gendomain.GenerationParams {
	maxTokens := int32(s.cfg.MaxTokens)
	temp := float32(s.cfg.Temperature)
	rfType := "json_object"

	return &gendomain.GenerationParams{
		MaxTokens:      &maxTokens,
		Temperature:    &temp,
		ResponseFormat: &gendomain.ResponseFormat{Type: rfType},
	}
}

const agentStepSystemPromptText = `You are an autonomous coding agent for the Coder editor.
Respond with a single JSON object only (no markdown fences):
{"finish":boolean,"summary":string,"calls":[{"tool":string,"id":string,"args":object}]}
Available tools: list_dir, read_file, glob_search, search_content, apply_patch, create_file, run_command.
If the goal is complete, set finish=true and calls=[].
If more work is needed, set finish=false and list tool calls with unique id fields like "call-1".
When a previous tool call failed (stderr, non-zero exit code), analyze the error and retry with a corrected approach.`

func (s *AgentService) buildAgentStepUserPrompt(req domain.AgentStepRequest) string {
	var b strings.Builder
	if g := strings.TrimSpace(req.Goal); g != "" {
		fmt.Fprintf(&b, "Goal: %s\n", g)
	}

	if sid := strings.TrimSpace(req.SessionID); sid != "" {
		fmt.Fprintf(&b, "Session: %s\n", sid)
		if s != nil && s.sessions != nil {
			if hint := s.sessions.ContextHint(sid); hint != "" {
				fmt.Fprintf(&b, "%s\n", hint)
			}
		}
	}

	if len(req.Context) > 0 {
		ctxJSON, _ := json.Marshal(req.Context)
		fmt.Fprintf(&b, "Context: %s\n", string(ctxJSON))
	}

	if obs := FormatObservations(req.Observations); obs != "" {
		b.WriteString(obs)
	}

	if b.Len() == 0 {
		return "Continue the agent session."
	}

	return b.String()
}

func parseAgentStepLLMOutput(raw string) (domain.AgentStepResponse, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var resp domain.AgentStepResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return domain.AgentStepResponse{}, err
	}

	if resp.Calls == nil {
		resp.Calls = []domain.AgentToolCall{}
	}

	return resp, nil
}
