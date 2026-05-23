package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/magomedcoder/gen/pkg/llmrunner"
	"gopkg.in/yaml.v3"
)

type GenerateConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

type ChatConfig struct {
	TimeoutSeconds int            `yaml:"timeout_seconds"`
	Generate       GenerateConfig `yaml:"generate"`
}

type AgentConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

type RunnerHintsConfig struct {
	MaxContextTokens        int `yaml:"max_context_tokens"`
	LLMHistoryMaxMessages   int `yaml:"llm_history_max_messages"`
	MaxToolInvocationRounds int `yaml:"max_tool_invocation_rounds"`
}

type RunnerConfig struct {
	Addr    string            `yaml:"addr"`
	Name    string            `yaml:"name"`
	Enabled *bool             `yaml:"enabled"`
	Hints   RunnerHintsConfig `yaml:"hints"`
}

type ContextConfig struct {
	TokenBudget int `yaml:"token_budget"`
}

type Config struct {
	Host    string         `yaml:"host"`
	Port    int            `yaml:"port"`
	Runners []RunnerConfig `yaml:"runners"`
	APIKey  string         `yaml:"api_key"`
	Chat    ChatConfig     `yaml:"chat"`
	Agent   AgentConfig    `yaml:"agent"`
	Context ContextConfig  `yaml:"context"`

	listenOverride string
}

func Load(path string) (*Config, error) {
	c := &Config{}

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла %s: %w", path, err)
		}

		if err := yaml.Unmarshal(data, c); err != nil {
			return nil, fmt.Errorf("ошибка парсинга конфигурационного файла %s: %w", path, err)
		}
	}

	return c, nil
}

func (c *Config) ChatTimeoutSeconds() int32 {
	if c == nil || c.Chat.TimeoutSeconds <= 0 {
		return 300
	}
	return int32(c.Chat.TimeoutSeconds)
}

func (c *Config) ContextTokenBudget() int {
	if c == nil || c.Context.TokenBudget <= 0 {
		return 8192
	}
	return c.Context.TokenBudget
}

func (c *Config) ListenAddr() string {
	if c.listenOverride != "" {
		return c.listenOverride
	}

	return net.JoinHostPort(strings.TrimSpace(c.Host), strconv.Itoa(c.Port))
}

func (c *Config) RunnerStates() []llmrunner.RunnerState {
	if c == nil {
		return nil
	}

	states := make([]llmrunner.RunnerState, 0, len(c.Runners))
	for _, r := range c.Runners {
		addr := strings.TrimSpace(r.Addr)
		if addr == "" {
			continue
		}

		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}

		var hints *llmrunner.RunnerCoreHints
		if h := r.Hints; h.MaxContextTokens > 0 || h.LLMHistoryMaxMessages > 0 || h.MaxToolInvocationRounds > 0 {
			hints = &llmrunner.RunnerCoreHints{
				MaxContextTokens:        h.MaxContextTokens,
				LLMHistoryMaxMessages:   h.LLMHistoryMaxMessages,
				MaxToolInvocationRounds: h.MaxToolInvocationRounds,
			}
		}

		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = addr
		}

		states = append(states, llmrunner.RunnerState{
			Address: addr,
			Name:    name,
			Enabled: enabled,
			Hints:   hints,
		})
	}

	return states
}

func (c *Config) AuthEnabled() bool {
	return c != nil && strings.TrimSpace(c.APIKey) != ""
}
