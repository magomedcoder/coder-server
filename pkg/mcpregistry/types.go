package mcpregistry

type ServerConfig struct {
	ID             int64
	Name           string
	Enabled        *bool
	Transport      string
	URL            string
	Headers        map[string]string
	TimeoutSeconds int32
}

type ToolInfo struct {
	Alias          string `json:"alias"`
	ServerID       int64  `json:"server_id"`
	ServerName     string `json:"server_name"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	ParametersJSON string `json:"-"`
}

type CallRequest struct {
	Tool      string         `json:"tool,omitempty"`
	ServerID  int64          `json:"server_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
