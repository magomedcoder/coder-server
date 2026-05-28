package domain

type MCPToolInfo struct {
	Alias       string `json:"alias"`
	ServerID    int64  `json:"server_id"`
	ServerName  string `json:"server_name"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type MCPToolsResponse struct {
	Tools []MCPToolInfo `json:"tools"`
}

type MCPCallRequest struct {
	Tool      string         `json:"tool,omitempty"`
	ServerID  int64          `json:"server_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type MCPCallResponse struct {
	Result string `json:"result"`
}

type IndexGraphRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Path        string `json:"path,omitempty"`
	Symbol      string `json:"symbol,omitempty"`
}

type IndexGraphResponse struct {
	Path       string   `json:"path,omitempty"`
	Symbol     string   `json:"symbol,omitempty"`
	Imports    []string `json:"imports,omitempty"`
	ImportedBy []string `json:"imported_by,omitempty"`
	ChunkIDs   []string `json:"chunk_ids,omitempty"`
}

type TestSuggestRequest struct {
	Path     string `json:"path,omitempty"`
	Language string `json:"language,omitempty"`
	Source   string `json:"source"`
	Error    string `json:"error,omitempty"`
}

type TestSuggestResponse struct {
	Summary  string `json:"summary"`
	TestCode string `json:"test_code"`
}
