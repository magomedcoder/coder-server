package domain

type ChatSession struct {
	SessionID      string   `json:"session_id,omitempty"`
	StopSequences  []string `json:"stop_sequences,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	TopK           *int     `json:"top_k,omitempty"`
	TopP           *float64 `json:"top_p,omitempty"`
	MCPEnabled     *bool    `json:"mcp_enabled,omitempty"`
	MCPServerIDs   []int64  `json:"mcp_server_ids,omitempty"`
	TimeoutSeconds *int     `json:"timeout_seconds,omitempty"`
}

type AgentRunRequest struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd,omitempty"`
}

type AgentRunResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type QueueJobStatus struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Kind      string `json:"kind,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Result    any    `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

type QueueStatsResponse struct {
	InFlight      int   `json:"in_flight"`
	Capacity      int   `json:"capacity"`
	PendingJobs   int   `json:"pending_jobs"`
	CompletedJobs int64 `json:"completed_jobs"`
}
