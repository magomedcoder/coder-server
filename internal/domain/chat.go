package domain

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	RequestID *string         `json:"request_id,omitempty"`
	Stream    *bool           `json:"stream"`
	System    *string         `json:"system"`
	Messages  []ChatMessage   `json:"messages"`
	Editor    *EditorContext  `json:"editor,omitempty"`
	Context   *ChatContext    `json:"context,omitempty"`
	Search    *ChatSearch     `json:"search,omitempty"`
	Session   *ChatSession    `json:"session,omitempty"`
	Generate  *GenerateParams `json:"generate,omitempty"`
}

type ChatSearch struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"query,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

type ChatResponse struct {
	Message ChatMessage `json:"message"`
	Finish  string      `json:"finish"`
	Usage   *TokenUsage `json:"usage,omitempty"`
}

type TokenUsage struct {
	PromptTokens     int32 `json:"prompt_tokens,omitempty"`
	CompletionTokens int32 `json:"completion_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens,omitempty"`
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
	RequestID    string                 `json:"request_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	Goal         string                 `json:"goal,omitempty"`
	Context      map[string]any         `json:"context,omitempty"`
	Observations []AgentStepObservation `json:"observations,omitempty"`
}

type HealthResponse struct {
	OK           bool               `json:"ok"`
	Runner       *HealthRunnerInfo  `json:"runner,omitempty"`
	Capabilities *ModelCapabilities `json:"capabilities,omitempty"`
}

type HealthRunnerInfo struct {
	Connected   bool   `json:"connected"`
	ModelLoaded bool   `json:"model_loaded"`
	Model       string `json:"model,omitempty"`
	Address     string `json:"address,omitempty"`
}

type ModelCapabilities struct {
	MaxContextTokens int  `json:"max_context_tokens,omitempty"`
	JSONMode         bool `json:"json_mode"`
	Embeddings       bool `json:"embeddings,omitempty"`
	Tools            bool `json:"tools,omitempty"`
	MCPServers       int  `json:"mcp_servers,omitempty"`
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
	Step    int             `json:"step,omitempty"`
	Blocked []string        `json:"blocked,omitempty"`
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

func NewErrorResponse(code, message string) ErrorResponse {
	var resp ErrorResponse
	resp.Error.Code = code
	resp.Error.Message = message
	return resp
}
