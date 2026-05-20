package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/toolloop"
)

type memStore struct {
	mu   sync.Mutex
	msgs []*domain.Message
	next int64
}

func (m *memStore) Create(_ context.Context, msg *domain.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	msg.Id = m.next
	m.msgs = append(m.msgs, msg)
	return nil
}

type scriptedLLM struct {
	rounds []string
	n      int
}

func (s *scriptedLLM) CheckConnection(context.Context) (bool, error) {
	return true, nil
}

func (s *scriptedLLM) GetModels(context.Context) ([]string, error) {
	return nil, nil
}

func (s *scriptedLLM) Embed(context.Context, string, string) ([]float32, error) {
	return nil, fmt.Errorf("n/a")
}

func (s *scriptedLLM) EmbedBatch(context.Context, string, []string) ([][]float32, error) {
	return nil, fmt.Errorf("n/a")
}

func (s *scriptedLLM) SendMessage(context.Context, int64, string, []*domain.Message, []string, int32, *domain.GenerationParams) (chan domain.LLMStreamChunk, error) {
	return nil, fmt.Errorf("n/a")
}

func (s *scriptedLLM) SendMessageOnRunner(_ context.Context, _ string, _ int64, _ string, _ []*domain.Message, _ []string, _ int32, _ *domain.GenerationParams) (chan domain.LLMStreamChunk, error) {
	s.n++
	if s.n > len(s.rounds) {
		return nil, fmt.Errorf("exhausted")
	}

	text := s.rounds[s.n-1]
	ch := make(chan domain.LLMStreamChunk, 1)
	ch <- domain.LLMStreamChunk{
		Content: text,
	}

	close(ch)

	return ch, nil
}

type stubExecutor struct{}

func (stubExecutor) Execute(context.Context, toolloop.ExecutableToolCall, *domain.GenerationParams, *toolloop.LoopEnv) (string, error) {
	return `{"ok":true}`, nil
}

func TestRun_finalAnswerNoTools(t *testing.T) {
	store := &memStore{}
	llm := &scriptedLLM{
		rounds: []string{"Привет мир"},
	}
	ch, err := Run(context.Background(), Config{
		SessionID:      1,
		RunnerAddr:     "127.0.0.1:1",
		Model:          "m",
		LLM:            llm,
		InitialHistory: []*domain.Message{domain.NewMessage(1, "hi", domain.MessageRoleUser)},
		GenParams: &domain.GenerationParams{
			Tools: []domain.Tool{
				{
					Name: "web_search",
				},
			},
		},
		MaxRounds: 4,
		Executor:  stubExecutor{},
		Store:     store,
	})
	if err != nil {
		t.Fatal(err)
	}

	var text string
	for c := range ch {
		text += c.Text
	}

	if !strings.Contains(text, "Hello") && !strings.Contains(text, "world") {
		t.Fatalf("stream: %q", text)
	}

	if len(store.msgs) != 1 || store.msgs[0].Role != domain.MessageRoleAssistant {
		t.Fatalf("messages: %+v", store.msgs)
	}
}

func TestRun_toolRoundThenAnswer(t *testing.T) {
	patch := `[{"tool_name":"apply_markdown_patch","parameters":{"path":"a.md","operations_json":"[]"}}]`
	store := &memStore{}
	llm := &scriptedLLM{
		rounds: []string{
			patch, "Done after tool",
		},
	}
	ch, err := Run(context.Background(), Config{
		SessionID:      2,
		RunnerAddr:     "127.0.0.1:1",
		Model:          "m",
		LLM:            llm,
		InitialHistory: []*domain.Message{domain.NewMessage(2, "edit", domain.MessageRoleUser)},
		GenParams: &domain.GenerationParams{
			Tools: []domain.Tool{
				{
					Name:           "apply_markdown_patch",
					ParametersJSON: `{}`,
				},
			},
		},
		MaxRounds: 5,
		Executor:  stubExecutor{},
		Store:     store,
	})

	if err != nil {
		t.Fatal(err)
	}

	for range ch {

	}

	if len(store.msgs) < 3 {
		t.Fatalf("expected assistant+tool+assistant, got %d", len(store.msgs))
	}
}
