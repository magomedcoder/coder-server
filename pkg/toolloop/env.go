package toolloop

import (
	"context"
	"encoding/json"

	"github.com/magomedcoder/gen/pkg/domain"
)

type CohereActionRow struct {
	ToolName   string          `json:"tool_name"`
	Parameters json.RawMessage `json:"parameters"`
}

type ExecutableToolCall struct {
	RequestedName string
	ResolvedName  string
	Parameters    json.RawMessage
}

type loopEnvKey struct{}

type LoopEnv struct {
	RunnerAddr             string
	ResolvedModel          string
	StopSequences          []string
	TimeoutSeconds         int32
	SamplingGen            *domain.GenerationParams
	UserTurnHasVisionImage bool
}

func WithLoopEnv(ctx context.Context, env *LoopEnv) context.Context {
	if env == nil {
		return ctx
	}

	return context.WithValue(ctx, loopEnvKey{}, env)
}

func LoopEnvFrom(ctx context.Context) *LoopEnv {
	v, _ := ctx.Value(loopEnvKey{}).(*LoopEnv)
	return v
}
