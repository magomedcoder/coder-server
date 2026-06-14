package domain

type CompleteRequest struct {
	Path         string          `json:"path"`
	Language     string          `json:"language,omitempty"`
	Prefix       string          `json:"prefix"`
	Suffix       string          `json:"suffix"`
	CursorLine   *int            `json:"cursor_line,omitempty"`
	CursorColumn *int            `json:"cursor_column,omitempty"`
	Generate     *GenerateParams `json:"generate,omitempty"`
}

type CompleteResponse struct {
	Text string `json:"text"`
}

type EditRequest struct {
	Path               string          `json:"path"`
	Language           string          `json:"language,omitempty"`
	Selection          string          `json:"selection"`
	Instruction        string          `json:"instruction"`
	SelectionStartLine *int            `json:"selection_start_line,omitempty"`
	SelectionStartCol  *int            `json:"selection_start_col,omitempty"`
	SelectionEndLine   *int            `json:"selection_end_line,omitempty"`
	SelectionEndCol    *int            `json:"selection_end_col,omitempty"`
	Generate           *GenerateParams `json:"generate,omitempty"`
}

type EditResponse struct {
	Text string `json:"text"`
}
