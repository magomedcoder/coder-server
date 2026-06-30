package domain

import "context"

type ResponseFormat struct {
	Type   string
	Schema *string
}

type Tool struct {
	Name           string
	Description    string
	ParametersJSON string
}

type GenerationParams struct {
	Temperature       *float32
	MaxTokens         *int32
	TopK              *int32
	TopP              *float32
	EnableThinking    *bool
	ResponseFormat    *ResponseFormat
	Tools             []Tool
	ChatTemplateJinja string
	RenderedPrompt    string
}

type LLMRepository interface {
	CheckConnection(ctx context.Context) (bool, error)

	GetModels(ctx context.Context) ([]string, error)

	SendMessage(
		ctx context.Context,
		messages []*Message,
		stopSequences []string,
		timeoutSeconds int32,
		genParams *GenerationParams,
	) (chan LLMStreamChunk, error)

	SendMessageOnRunner(
		ctx context.Context,
		runnerListenAddr string,
		messages []*Message,
		stopSequences []string,
		timeoutSeconds int32,
		genParams *GenerationParams,
	) (chan LLMStreamChunk, error)

	Embed(ctx context.Context, text string) ([]float32, error)

	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
