package agent

import (
	"context"

	"github.com/magomedcoder/gen/pkg/chatstream"
	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/toolloop"
)

type ToolExecutor interface {
	Execute(ctx context.Context, call toolloop.ExecutableToolCall, gp *domain.GenerationParams, env *toolloop.LoopEnv) (result string, err error)
}

type MessageStore interface {
	Create(ctx context.Context, msg *domain.Message) error
}

type HistoryCapper interface {
	Cap(ctx context.Context, history []*domain.Message, addedCount int) (capped []*domain.Message, trimmed bool, err error)
}

type Config struct {
	SessionID  int64
	RunnerAddr string
	Model      string

	LLM            domain.LLMRepository
	InitialHistory []*domain.Message
	StopSequences  []string
	TimeoutSeconds int32
	GenParams      *domain.GenerationParams

	MaxRounds int

	SamplingModel string

	Executor ToolExecutor

	ToolDisplayName func(ctx context.Context, resolvedName, requestedName string) string

	Store MessageStore
	Cap   HistoryCapper

	Preface []chatstream.ChatStreamChunk
}
