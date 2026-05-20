package domain

type StreamTokenUsage struct {
	PromptTokens     int32
	CompletionTokens int32
	TotalTokens      int32
}

type LLMStreamChunk struct {
	Content          string
	ReasoningContent string
	Usage            *StreamTokenUsage
}
