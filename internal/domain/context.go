package domain

type WorkspaceContext struct {
	Root   string `json:"root,omitempty"`
	Branch string `json:"branch,omitempty"`
	Name   string `json:"name,omitempty"`
}

type SelectionContext struct {
	Path      string `json:"path,omitempty"`
	Language  string `json:"language,omitempty"`
	Text      string `json:"text,omitempty"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
}

type ContextSnippet struct {
	Path     string `json:"path,omitempty"`
	Language string `json:"language,omitempty"`
	Content  string `json:"content,omitempty"`
	Source   string `json:"source,omitempty"`
}

type ChatContext struct {
	Workspace *WorkspaceContext `json:"workspace,omitempty"`
	Selection *SelectionContext `json:"selection,omitempty"`
	Snippets  []ContextSnippet  `json:"snippets,omitempty"`
	Extra     map[string]any    `json:"extra,omitempty"`
}
