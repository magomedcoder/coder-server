package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/magomedcoder/coder-server/internal/config"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
)

type AgentService struct {
	llm            *LLMRunnerService
	cfg            config.AgentConfig
	timeoutSeconds int32
	tokenBudget    int
}

func NewAgentService(llm *LLMRunnerService, cfg *config.Config) *AgentService {
	return &AgentService{
		llm:            llm,
		cfg:            cfg.Agent,
		timeoutSeconds: cfg.ChatTimeoutSeconds(),
		tokenBudget:    cfg.ContextTokenBudget(),
	}
}

func (s *AgentService) contextTokenBudget() int {
	if s == nil || s.tokenBudget <= 0 {
		return 8192
	}
	return s.tokenBudget
}

func (s *AgentService) Step(ctx context.Context, req domain.AgentStepRequest) (domain.AgentStepResponse, error) {
	system := agentStepSystemPrompt()
	user := buildAgentStepUserPrompt(req)

	messages := mapper.RunnerMessages(system, []domain.ChatMessage{
		{
			Role:    "user",
			Content: user,
		},
	}, nil, nil, s.contextTokenBudget())

	genParams := s.agentGenerationParams()

	raw, err := s.llm.CollectMessage(ctx, messages, nil, s.timeoutSeconds, genParams)
	if err != nil {
		return domain.AgentStepResponse{}, err
	}

	parsed, err := parseAgentStepLLMOutput(raw)
	if err != nil {
		return domain.AgentStepResponse{
			Finish:  true,
			Summary: strings.TrimSpace(raw),
			Calls:   nil,
		}, nil
	}

	return parsed, nil
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
If more work is needed, set finish=false and list tool calls with unique id fields like "call-1".`

func agentStepSystemPrompt() string {
	return agentStepSystemPromptText
}

func buildAgentStepUserPrompt(req domain.AgentStepRequest) string {
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
