package toolloop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/magomedcoder/coder-server/pkg/domain"
	"strings"
	"time"
)

const (
	MaxToolResultRunes     = 8000
	MinToolExecSeconds     = 30
	MaxToolExecSeconds     = 300
	DefaultToolExecSeconds = 120
	DefaultToolLoopRounds  = 12
	MaxToolLoopRoundsCap   = 128
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
	SamplingModel          string
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

func ToolExecutionDuration(sessionTimeoutSec int32) time.Duration {
	s := int64(sessionTimeoutSec)
	if s <= 0 {
		s = DefaultToolExecSeconds
	}

	if s < MinToolExecSeconds {
		s = MinToolExecSeconds
	}

	if s > MaxToolExecSeconds {
		s = MaxToolExecSeconds
	}

	return time.Duration(s) * time.Second
}

func RunFnWithContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return fn()
	}

	type result struct {
		val T
		err error
	}

	ch := make(chan result, 1)
	go func() {
		v, err := fn()
		ch <- result{v, err}
	}()

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case r := <-ch:
		return r.val, r.err
	}
}

func ErrorToolMessage(call ExecutableToolCall, err error, partialResult string, deadlineExceeded bool) string {
	var b strings.Builder
	b.WriteString("Статус: error выполнения инструмента.\n")
	b.WriteString("Твоя задача: кратко и по-русски объясни пользователю, что пошло не так и что можно сделать (повторить request, проверить права, уточнить параметры).\n\n")
	if deadlineExceeded {
		b.WriteString("Причина: истёк timeout ожидания responseа инструмента.\n")
	} else if err != nil {
		b.WriteString("Причина (технически): ")
		b.WriteString(strings.TrimSpace(err.Error()))
		b.WriteByte('\n')
	}

	b.WriteString("Запрошенное имя инструмента: ")
	b.WriteString(strings.TrimSpace(call.RequestedName))
	b.WriteString("\nВнутреннее имя: ")
	b.WriteString(strings.TrimSpace(call.ResolvedName))
	b.WriteByte('\n')
	pr := strings.TrimSpace(partialResult)
	errText := ""
	if err != nil {
		errText = strings.TrimSpace(err.Error())
	}

	if pr != "" && pr != errText {
		b.WriteString("\nДополнительно от сервера или среды выполнения:\n")
		b.WriteString(pr)
		b.WriteByte('\n')
	}

	return TruncateToolResult(strings.TrimSpace(b.String()))
}

func MustStringField(m map[string]json.RawMessage, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("отсутствует поле %q", key)
	}

	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", fmt.Errorf("поле %q: ожидается строка", key)
	}

	return strings.TrimSpace(s), nil
}

func OptionalStringField(m map[string]json.RawMessage, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}

	var s string
	_ = json.Unmarshal(v, &s)
	return strings.TrimSpace(s)
}

func OptionalInt64Field(m map[string]json.RawMessage, key string) (int64, bool, error) {
	v, ok := m[key]
	if !ok {
		return 0, false, nil
	}

	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, false, err
	}

	return int64(f), true, nil
}

func OptionalInt32Field(m map[string]json.RawMessage, key string) (int32, bool, error) {
	v, ok := m[key]
	if !ok {
		return 0, false, nil
	}

	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, false, err
	}

	return int32(f), true, nil
}

func FilterExecutableToolRows(rows []CohereActionRow) []CohereActionRow {
	var out []CohereActionRow
	for _, r := range rows {
		if !IsDirectAnswerTool(r.ToolName) {
			out = append(out, r)
		}
	}

	return out
}
