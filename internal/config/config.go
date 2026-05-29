package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/magomedcoder/gen/pkg/llmrunner"
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
	Enabled           bool   `yaml:"enabled"`
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
	Addr    string            `yaml:"addr"`
	Name    string            `yaml:"name"`
	Enabled *bool             `yaml:"enabled"`
	Hints   RunnerHintsConfig `yaml:"hints"`
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
	Structured bool `yaml:"structured"`
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

	c.applyDefaults()
	return c, nil
}

func (c *Config) applyDefaults() {
	if c == nil {
		return
	}

	if c.Host == "" {
		c.Host = "127.0.0.1"
	}

	if c.Port <= 0 {
		c.Port = 8000
	}

	if c.Chat.TimeoutSeconds <= 0 {
		c.Chat.TimeoutSeconds = 300
	}

	if c.Chat.Generate.MaxTokens <= 0 {
		c.Chat.Generate.MaxTokens = 1024
	}

	if c.Chat.Generate.Temperature <= 0 {
		c.Chat.Generate.Temperature = 0.2
	}

	if c.Chat.HistoryMaxMessages <= 0 {
		c.Chat.HistoryMaxMessages = 40
	}

	if c.Agent.MaxTokens <= 0 {
		c.Agent.MaxTokens = 2048
	}

	if c.Agent.Temperature <= 0 {
		c.Agent.Temperature = 0.1
	}

	if c.Agent.MaxSteps <= 0 {
		c.Agent.MaxSteps = 30
	}

	if len(c.Agent.BlockedCommands) == 0 {
		c.Agent.BlockedCommands = defaultBlockedCommands()
	}

	if len(c.Agent.AllowedCommands) == 0 {
		c.Agent.AllowedCommands = defaultAllowedCommands()
	}

	if c.Agent.Sandbox.MaxOutputBytes <= 0 {
		c.Agent.Sandbox.MaxOutputBytes = 65536
	}

	if c.Agent.Sandbox.CommandTimeoutSec <= 0 {
		c.Agent.Sandbox.CommandTimeoutSec = 120
	}

	if c.Context.TokenBudget <= 0 {
		c.Context.TokenBudget = 8192
	}

	if c.Context.ScanSecrets == nil {
		v := true
		c.Context.ScanSecrets = &v
	}

	if c.Reliability.RunnerRetries <= 0 {
		c.Reliability.RunnerRetries = 2
	}

	if c.Reliability.CircuitBreakerFailures <= 0 {
		c.Reliability.CircuitBreakerFailures = 3
	}
	if c.Reliability.CircuitBreakerCooldownSecs <= 0 {
		c.Reliability.CircuitBreakerCooldownSecs = 30
	}

	if c.Reliability.MaxConcurrentRequests <= 0 {
		c.Reliability.MaxConcurrentRequests = 8
	}

	if c.Reliability.QueueWaitSeconds <= 0 {
		c.Reliability.QueueWaitSeconds = 60
	}

	if c.Reliability.PersistentQueueMax <= 0 {
		c.Reliability.PersistentQueueMax = 256
	}

	if c.RateLimit.RequestsPerMinute <= 0 {
		c.RateLimit.RequestsPerMinute = 120
	}

	if c.SSE.BufferTTLSeconds <= 0 {
		c.SSE.BufferTTLSeconds = 300
	}

	if c.Index.MaxChunksPerWorkspace <= 0 {
		c.Index.MaxChunksPerWorkspace = 10000
	}

	if c.Index.SearchWorkers <= 0 {
		c.Index.SearchWorkers = 4
	}

	if strings.TrimSpace(c.Index.Qdrant.CollectionPrefix) == "" {
		c.Index.Qdrant.CollectionPrefix = "coder"
	}

	if c.Idempotency.TTLSeconds <= 0 {
		c.Idempotency.TTLSeconds = 300
	}

	if c.Security.ModerationEnabled == nil {
		v := true
		c.Security.ModerationEnabled = &v
	}

	if c.Cache.PromptPrefixEntries <= 0 {
		c.Cache.PromptPrefixEntries = 256
	}

	c.ensureDefaultRunners()
}

func defaultBlockedCommands() []string {
	return []string{
		"rm -rf /",
		"mkfs.",
		":(){ :|:& };:",
		"curl | sh",
		"wget | sh",
		"curl | bash",
		"wget | bash",
		"sudo ",
		"chmod 777 /",
		"dd if=",
	}
}

func defaultAllowedCommands() []string {
	return []string{
		"cargo", "go", "npm", "yarn", "pnpm", "python", "python3", "pytest", "make", "rustc", "node", "deno", "bash", "sh",
	}
}

func (c *Config) ensureDefaultRunners() {
	if c == nil || len(c.Runners) > 0 {
		return
	}
	c.Runners = []RunnerConfig{{
		Addr: "127.0.0.1:50052",
		Name: "local",
	}}
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

func (c *Config) ContextScanSecrets() bool {
	if c == nil || c.Context.ScanSecrets == nil {
		return true
	}

	return *c.Context.ScanSecrets
}

func (c *Config) SSEBufferTTL() time.Duration {
	if c == nil || c.SSE.BufferTTLSeconds <= 0 {
		return 5 * time.Minute
	}

	return time.Duration(c.SSE.BufferTTLSeconds) * time.Second
}

func (c *Config) CircuitBreakerCooldown() time.Duration {
	if c == nil || c.Reliability.CircuitBreakerCooldownSecs <= 0 {
		return 30 * time.Second
	}

	return time.Duration(c.Reliability.CircuitBreakerCooldownSecs) * time.Second
}

func (c *Config) QueueWaitTimeout() time.Duration {
	if c == nil || c.Reliability.QueueWaitSeconds <= 0 {
		return 60 * time.Second
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

func (c *Config) RateLimitEnabled() bool {
	return c != nil && c.RateLimit.RequestsPerMinute > 0
}

func (c *Config) MaxIndexChunks() int {
	if c == nil || c.Index.MaxChunksPerWorkspace <= 0 {
		return 10000
	}

	return c.Index.MaxChunksPerWorkspace
}

func (c *Config) IdempotencyTTL() time.Duration {
	if c == nil || c.Idempotency.TTLSeconds <= 0 {
		return 5 * time.Minute
	}

	return time.Duration(c.Idempotency.TTLSeconds) * time.Second
}

func (c *Config) ModerationEnabled() bool {
	if c == nil || c.Security.ModerationEnabled == nil {
		return true
	}

	return *c.Security.ModerationEnabled
}

func (c *Config) PromptCacheEntries() int {
	if c == nil || c.Cache.PromptPrefixEntries <= 0 {
		return 256
	}

	return c.Cache.PromptPrefixEntries
}

func (c *Config) SearchWorkers() int {
	if c == nil || c.Index.SearchWorkers <= 0 {
		return 4
	}

	return c.Index.SearchWorkers
}

func (c *Config) HistoryMaxMessages() int {
	if c == nil {
		return 40
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

	if c.Chat.HistoryMaxMessages > 0 {
		return c.Chat.HistoryMaxMessages
	}

	return 40
}

func (c *Config) PersistentQueueEnabled() bool {
	return c != nil && strings.TrimSpace(c.Reliability.PersistentQueuePath) != ""
}

func (c *Config) QdrantEnabled() bool {
	return c != nil && strings.TrimSpace(c.Index.Qdrant.URL) != ""
}

func (c *Config) SandboxEnabled() bool {
	return c != nil && c.Agent.Sandbox.Enabled && strings.TrimSpace(c.Agent.Sandbox.WorkspaceRoot) != ""
}
