package domain

type IndexChunk struct {
	ID         string   `json:"id"`
	Path       string   `json:"path,omitempty"`
	Language   string   `json:"language,omitempty"`
	Content    string   `json:"content"`
	Symbol     string   `json:"symbol,omitempty"`
	SymbolType string   `json:"symbol_type,omitempty"`
	Imports    []string `json:"imports,omitempty"`
}

type IndexSyncRequest struct {
	WorkspaceID string       `json:"workspace_id"`
	Upsert      []IndexChunk `json:"upsert,omitempty"`
	Delete      []string     `json:"delete,omitempty"`
}

type IndexSyncResponse struct {
	Chunks int `json:"chunks"`
}

type SearchRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"query"`
	Limit       int    `json:"limit,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

type SearchHit struct {
	ID       string   `json:"id"`
	Path     string   `json:"path,omitempty"`
	Language string   `json:"language,omitempty"`
	Symbol   string   `json:"symbol,omitempty"`
	Score    float64  `json:"score"`
	Snippet  string   `json:"snippet"`
	Related  []string `json:"related,omitempty"`
}

type SearchResponse struct {
	Hits []SearchHit `json:"hits"`
	Mode string      `json:"mode"`
}

type ModelsResponse struct {
	Models []string `json:"models"`
}
