package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/magomedcoder/coder-server/pkg/contextbudget"
	"github.com/magomedcoder/coder-server/pkg/llmclient"
	"github.com/magomedcoder/coder-server/pkg/llmrunner"
	"github.com/magomedcoder/coder-server/pkg/mcpregistry"
	"gopkg.in/yaml.v3"
)

type GenerateConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

type ChatConfig struct {
	TimeoutSeconds     int            `yaml:"timeout_seconds"`
	HistoryMaxMessages int            `yaml:"history_max_messages"`
	Generate           GenerateConfig `yaml:"generate"`
}

type AgentConfig struct {
	MaxTokens       int                `yaml:"max_tokens"`
	Temperature     float64            `yaml:"temperature"`
	MaxSteps        int                `yaml:"max_steps"`
	AllowedPaths    []string           `yaml:"allowed_paths"`
	BlockedCommands []string           `yaml:"blocked_commands"`
	AllowedCommands []string           `yaml:"allowed_commands"`
	Sandbox         AgentSandboxConfig `yaml:"sandbox"`
}

type AgentSandboxConfig struct {
	Enabled           *bool  `yaml:"enabled"`
	WorkspaceRoot     string `yaml:"workspace_root"`
	MaxOutputBytes    int    `yaml:"max_output_bytes"`
	CommandTimeoutSec int    `yaml:"command_timeout_seconds"`
}

type RunnerHintsConfig struct {
	MaxContextTokens        int `yaml:"max_context_tokens"`
	LLMHistoryMaxMessages   int `yaml:"llm_history_max_messages"`
	MaxToolInvocationRounds int `yaml:"max_tool_invocation_rounds"`
}

type RunnerConfig struct {
	Addr          string            `yaml:"addr"`
	Name          string            `yaml:"name"`
	SelectedModel string            `yaml:"selected_model"`
	Enabled       *bool             `yaml:"enabled"`
	Hints         RunnerHintsConfig `yaml:"hints"`
}

type ContextConfig struct {
	TokenBudget int   `yaml:"token_budget"`
	ScanSecrets *bool `yaml:"scan_secrets"`
}

type ReliabilityConfig struct {
	RunnerRetries              int    `yaml:"runner_retries"`
	CircuitBreakerFailures     int    `yaml:"circuit_breaker_failures"`
	CircuitBreakerCooldownSecs int    `yaml:"circuit_breaker_cooldown_seconds"`
	MaxConcurrentRequests      int    `yaml:"max_concurrent_requests"`
	QueueWaitSeconds           int    `yaml:"queue_wait_seconds"`
	PersistentQueuePath        string `yaml:"persistent_queue_path"`
	PersistentQueueMax         int    `yaml:"persistent_queue_max"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
}

type SSEConfig struct {
	BufferTTLSeconds int `yaml:"buffer_ttl_seconds"`
}

type LoggingConfig struct {
	Structured *bool `yaml:"structured"`
}

type QuotasConfig struct {
	MaxTokensPerDay int64 `yaml:"max_tokens_per_day"`
}

type IndexConfig struct {
	MaxChunksPerWorkspace int          `yaml:"max_chunks_per_workspace"`
	SearchWorkers         int          `yaml:"search_workers"`
	Qdrant                QdrantConfig `yaml:"qdrant"`
}

type QdrantConfig struct {
	URL              string `yaml:"url"`
	CollectionPrefix string `yaml:"collection_prefix"`
	APIKey           string `yaml:"api_key"`
}

type MCPServerConfig struct {
	ID             int64             `yaml:"id"`
	Name           string            `yaml:"name"`
	Enabled        *bool             `yaml:"enabled"`
	Transport      string            `yaml:"transport"`
	URL            string            `yaml:"url"`
	Headers        map[string]string `yaml:"headers"`
	TimeoutSeconds int32             `yaml:"timeout_seconds"`
}

type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers"`
}

type IdempotencyConfig struct {
	TTLSeconds int `yaml:"ttl_seconds"`
}

type SecurityConfig struct {
	ModerationEnabled *bool `yaml:"moderation_enabled"`
}

type CacheConfig struct {
	PromptPrefixEntries int `yaml:"prompt_prefix_entries"`
}

type Config struct {
	Host        string            `yaml:"host"`
	Port        int               `yaml:"port"`
	Runners     []RunnerConfig    `yaml:"runners"`
	APIKey      string            `yaml:"api_key"`
	Chat        ChatConfig        `yaml:"chat"`
	Agent       AgentConfig       `yaml:"agent"`
	Context     ContextConfig     `yaml:"context"`
	Reliability ReliabilityConfig `yaml:"reliability"`
	RateLimit   RateLimitConfig   `yaml:"rate_limit"`
	SSE         SSEConfig         `yaml:"sse"`
	Logging     LoggingConfig     `yaml:"logging"`
	Quotas      QuotasConfig      `yaml:"quotas"`
	Index       IndexConfig       `yaml:"index"`
	Idempotency IdempotencyConfig `yaml:"idempotency"`
	Security    SecurityConfig    `yaml:"security"`
	Cache       CacheConfig       `yaml:"cache"`
	MCP         MCPConfig         `yaml:"mcp"`

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

	c.expandPaths()
	return c, nil
}

func (c *Config) expandPaths() {
	if c == nil {
		return
	}

	c.Agent.Sandbox.WorkspaceRoot = expandHome(c.Agent.Sandbox.WorkspaceRoot)
	c.Reliability.PersistentQueuePath = expandHome(c.Reliability.PersistentQueuePath)
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}

func (c *Config) ChatTimeoutSeconds() int32 {
	if c == nil {
		return 0
	}

	return int32(c.Chat.TimeoutSeconds)
}

func (c *Config) ContextTokenBudget() int {
	if c == nil {
		return 0
	}

	return c.Context.TokenBudget
}

func (c *Config) EffectiveContextTokenBudget(runnerMax int) int {
	if c == nil {
		return contextbudget.DefaultTokenBudget
	}

	return contextbudget.EffectiveBudget(c.Context.TokenBudget, runnerMax)
}

func (c *Config) ContextScanSecrets() bool {
	if c == nil || c.Context.ScanSecrets == nil {
		return false
	}

	return *c.Context.ScanSecrets
}

func (c *Config) SSEBufferTTL() time.Duration {
	if c == nil {
		return 0
	}

	return time.Duration(c.SSE.BufferTTLSeconds) * time.Second
}

func (c *Config) CircuitBreakerCooldown() time.Duration {
	if c == nil {
		return 0
	}

	return time.Duration(c.Reliability.CircuitBreakerCooldownSecs) * time.Second
}

func (c *Config) QueueWaitTimeout() time.Duration {
	if c == nil {
		return 0
	}

	return time.Duration(c.Reliability.QueueWaitSeconds) * time.Second
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
			Address:       addr,
			Name:          name,
			Enabled:       enabled,
			SelectedModel: strings.TrimSpace(r.SelectedModel),
			Hints:         hints,
		})
	}

	return states
}

func (c *Config) LLMClientOptions() llmclient.Options {
	if c == nil {
		return llmclient.Options{}
	}

	return llmclient.Options{
		RunnerStates: c.RunnerStates(),
		Reliability: llmclient.ReliabilityOptions{
			RunnerRetries:          c.Reliability.RunnerRetries,
			CircuitBreakerFailures: c.Reliability.CircuitBreakerFailures,
			CircuitBreakerCooldown: c.CircuitBreakerCooldown(),
			MaxConcurrentRequests:  c.Reliability.MaxConcurrentRequests,
			QueueWaitTimeout:       c.QueueWaitTimeout(),
		},
		SSEBufferTTL: c.SSEBufferTTL(),
	}
}

func (c *Config) MCPServerConfigs() []mcpregistry.ServerConfig {
	if c == nil {
		return nil
	}

	out := make([]mcpregistry.ServerConfig, 0, len(c.MCP.Servers))
	for _, s := range c.MCP.Servers {
		out = append(out, mcpregistry.ServerConfig{
			ID:             s.ID,
			Name:           s.Name,
			Enabled:        s.Enabled,
			Transport:      s.Transport,
			URL:            s.URL,
			Headers:        s.Headers,
			TimeoutSeconds: s.TimeoutSeconds,
		})
	}

	return out
}

func (c *Config) AuthEnabled() bool {
	return c != nil && strings.TrimSpace(c.APIKey) != ""
}

func (c *Config) RateLimitEnabled() bool {
	return c != nil && c.RateLimit.RequestsPerMinute > 0
}

func (c *Config) MaxIndexChunks() int {
	if c == nil {
		return 0
	}

	return c.Index.MaxChunksPerWorkspace
}

func (c *Config) IdempotencyTTL() time.Duration {
	if c == nil {
		return 0
	}

	return time.Duration(c.Idempotency.TTLSeconds) * time.Second
}

func (c *Config) ModerationEnabled() bool {
	if c == nil || c.Security.ModerationEnabled == nil {
		return false
	}

	return *c.Security.ModerationEnabled
}

func (c *Config) PromptCacheEntries() int {
	if c == nil {
		return 0
	}

	return c.Cache.PromptPrefixEntries
}

func (c *Config) SearchWorkers() int {
	if c == nil {
		return 0
	}

	return c.Index.SearchWorkers
}

func (c *Config) HistoryMaxMessages() int {
	if c == nil {
		return 0
	}

	for _, r := range c.Runners {
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		if !enabled {
			continue
		}
		if r.Hints.LLMHistoryMaxMessages > 0 {
			return r.Hints.LLMHistoryMaxMessages
		}
	}

	return c.Chat.HistoryMaxMessages
}

func (c *Config) PersistentQueueEnabled() bool {
	return c != nil && strings.TrimSpace(c.Reliability.PersistentQueuePath) != ""
}

func (c *Config) QdrantEnabled() bool {
	return c != nil && strings.TrimSpace(c.Index.Qdrant.URL) != ""
}

func (c *Config) SandboxEnabled() bool {
	if c == nil {
		return false
	}

	enabled := true
	if c.Agent.Sandbox.Enabled != nil {
		enabled = *c.Agent.Sandbox.Enabled
	}

	return enabled && strings.TrimSpace(c.Agent.Sandbox.WorkspaceRoot) != ""
}

func (c *Config) StructuredLogging() bool {
	if c == nil || c.Logging.Structured == nil {
		return false
	}

	return *c.Logging.Structured
}
