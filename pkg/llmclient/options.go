package llmclient

import (
	"time"

	"github.com/magomedcoder/coder-server/pkg/llmrunner"
)

type ReliabilityOptions struct {
	RunnerRetries          int
	CircuitBreakerFailures int
	CircuitBreakerCooldown time.Duration
	MaxConcurrentRequests  int
	QueueWaitTimeout       time.Duration
}

type Options struct {
	RunnerStates []llmrunner.RunnerState
	Reliability  ReliabilityOptions
	SSEBufferTTL time.Duration
	OnTokens     func(prompt, completion int32)
}
