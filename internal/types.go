package internal

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Stream   *bool           `json:"stream"`
	System   *string         `json:"system"`
	Messages []ChatMessage   `json:"messages"`
	Editor   *EditorContext  `json:"editor,omitempty"`
	Generate *GenerateParams `json:"generate,omitempty"`
}

type ChatResponse struct {
	Message ChatMessage `json:"message"`
	Finish  string      `json:"finish"`
}

type EditorContext struct {
	Path         string `json:"path,omitempty"`
	Language     string `json:"language,omitempty"`
	Snippet      string `json:"snippet,omitempty"`
	CursorLine   *int   `json:"cursor_line,omitempty"`
	CursorColumn *int   `json:"cursor_column,omitempty"`
}

type GenerateParams struct {
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type AgentStepRequest struct {
	SessionID    string                 `json:"session_id,omitempty"`
	Goal         string                 `json:"goal,omitempty"`
	Context      map[string]any         `json:"context,omitempty"`
	Observations []AgentStepObservation `json:"observations,omitempty"`
}

type AgentStepObservation struct {
	CallID string         `json:"call_id,omitempty"`
	Tool   string         `json:"tool,omitempty"`
	OK     bool           `json:"ok"`
	Result map[string]any `json:"result,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type AgentStepResponse struct {
	Finish  bool            `json:"finish"`
	Summary string          `json:"summary"`
	Calls   []AgentToolCall `json:"calls"`
}

type AgentToolCall struct {
	Tool string         `json:"tool"`
	ID   string         `json:"id"`
	Args map[string]any `json:"args"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
