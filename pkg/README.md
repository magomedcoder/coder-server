# lmpkg

Общая Go-библиотека оркестрации LLM: agent/tool-loop, MCP-клиент, промпты, RAG, доменные типы и gRPC-клиент к **lm-runner**.

Процесс **lm-runner** нужно **запустить отдельно**.

## Роли компонентов

```text
							gRPC (llmrunner.proto)
	ваш Go-сервис  -------------------------------> lm-runner (LLM)
			|					SendMessage, Embed, ...
        	|
        	|  tools/MCP/RAG/сессии - пакеты lmpkg/*
        	|
			---> lite-mcp-servers (опционально, отдельные процессы)
```

## go.mod стороннего проекта

```go
module test.test/test

go 1.26

require (
    github.com/magomedcoder/lmpkg v0.x.x
)
```

или clone https://github.com/magomedcoder/lmpkg.git

```go
replace github.com/magomedcoder/lmpkg => ../lmpkg
```

## Минимальное подключение к lm-runner

```go
import (
    "context"

    "github.com/magomedcoder/lmpkg/domain"
    "github.com/magomedcoder/lmpkg/llmrunner"
    "github.com/magomedcoder/lmpkg/toolloop"
)

func main() {
    ctx := context.Background()

    conn, err := llmrunner.ConnectFromAddresses(ctx, []string{"127.0.0.1:50052"}, true)
    if err != nil {
        panic(err)
    }
    defer conn.Pool.Close()

    llm := conn.LLM // domain.LLMRepository

    if err := conn.Pool.WarmModelOnRunner(ctx, "127.0.0.1:50052", "my-model"); err != nil {
        panic(err)
    }

    msgs := []*domain.Message{
        domain.NewMessage(1, "Вы - полезный помощник.", domain.MessageRoleSystem),
        domain.NewMessage(1, "Привет", domain.MessageRoleUser),
    }

    gp := &domain.GenerationParams{}

    ch, err := llm.SendMessageOnRunner(ctx, "127.0.0.1:50052", msgs, nil, 120, gp)
    if err != nil {
        panic(err)
    }

    var full string
    for c := range ch {
        full += c.Content
    }

    if blob := toolloop.ExtractToolActionBlob(full); blob != "" {
        // tool-loop parse -> execute -> снова llm.SendMessageOnRunner
        _ = blob
    }
}
```

## Пример: agent + tool-loop

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/magomedcoder/lmpkg/agent"
	"github.com/magomedcoder/lmpkg/domain"
	"github.com/magomedcoder/lmpkg/llmrunner"
	"github.com/magomedcoder/lmpkg/prompt"
	"github.com/magomedcoder/lmpkg/toolloop"
)

type memStore struct {
	msgs []*domain.Message
	id   int64
}

func (m *memStore) Create(_ context.Context, msg *domain.Message) error {
	m.id++
	msg.Id = m.id
	m.msgs = append(m.msgs, msg)
	return nil
}

type echoExecutor struct{}

func (echoExecutor) Execute(_ context.Context, call toolloop.ExecutableToolCall, _ *domain.GenerationParams, _ *toolloop.LoopEnv) (string, error) {
	return fmt.Sprintf(`{"tool":%q,"ok":true}`, call.ResolvedName), nil
}

func main() {
	runnerAddr := "lm-runner gRPC address"
	model := "model alias для LoadModel (обязательно)"

	ctx := context.Background()
	conn, err := llmrunner.ConnectFromAddresses(ctx, []string{*runnerAddr}, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "runner: %v\n", err)
		os.Exit(1)
	}
	defer conn.Pool.Close()

	if err := conn.Pool.WarmModelOnRunner(ctx, *runnerAddr, *model); err != nil {
		fmt.Fprintf(os.Stderr, "load model: %v\n", err)
		os.Exit(1)
	}

	sessionID := int64(1)
	tools := []domain.Tool{
		{
			Name:           "echo_tool",
			Description:    "Возвращает echo JSON",
			ParametersJSON: `{"type":"object"}`,
		},
	}
	sys := domain.NewMessage(sessionID, "Вы очень полезны.", domain.MessageRoleSystem)
	prompt.EnrichSystemMessage(sys, prompt.SystemToolsOptions{
		Tools: tools,
	})
	user := domain.NewMessage(sessionID, "Вызови echo_tool с пустыми parameters, затем ответь пользователю кратко.", domain.MessageRoleUser)

	gp := &domain.GenerationParams{
		Tools: tools,
	}
	store := &memStore{}

	ch, err := agent.Run(ctx, agent.Config{
		SessionID:      sessionID,
		RunnerAddr:     *runnerAddr,
		SelectedModel:  *model,
		LLM:            conn.LLM,
		InitialHistory: []*domain.Message{sys, user},
		TimeoutSeconds: 120,
		GenParams:      gp,
		MaxRounds:      6,
		Executor:       echoExecutor{},
		Store:          store,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "агент: %v\n", err)
		os.Exit(1)
	}

	for c := range ch {
		if c.Text != "" {
			fmt.Print(c.Text)
		}
	}
	fmt.Println()
	fmt.Fprintf(os.Stderr, "сообщений: %d\n", len(store.msgs))
}
```
