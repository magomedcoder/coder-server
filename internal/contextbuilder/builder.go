package contextbuilder

import (
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/security"
)

const approxCharsPerToken = 4

type Builder struct {
	tokenBudget int
	scanSecrets bool
}

func New(tokenBudget int, scanSecrets bool) *Builder {
	if tokenBudget <= 0 {
		tokenBudget = 8192
	}
	return &Builder{tokenBudget: tokenBudget, scanSecrets: scanSecrets}
}

func (b *Builder) Build(system string, editor *domain.EditorContext, ctx *domain.ChatContext) string {
	reserved := estimateTokens(system) + 512
	budget := max(b.tokenBudget-reserved, 256)

	var parts []contextPart

	if ctx != nil {
		if ws := workspacePrompt(ctx.Workspace); ws != "" {
			parts = append(parts, contextPart{
				priority: 10,
				text:     ws,
				tokens:   estimateTokens(ws),
			})
		}
		if sel := selectionPrompt(ctx.Selection); sel != "" {
			parts = append(parts, contextPart{
				priority: 0,
				text:     sel,
				tokens:   estimateTokens(sel),
			})
		}
		for _, sn := range ctx.Snippets {
			if p := snippetPrompt(sn); p != "" {
				priority := snippetPriority(sn.Source)
				parts = append(parts, contextPart{
					priority: priority,
					text:     p, tokens: estimateTokens(p),
				})
			}
		}
		if tree := treePrompt(ctx.Tree); tree != "" {
			parts = append(parts, contextPart{
				priority: 4,
				text:     tree,
				tokens:   estimateTokens(tree),
			})
		}
	}

	if ed := editorContextPrompt(editor); ed != "" {
		parts = append(parts, contextPart{
			priority: 1,
			text:     ed,
			tokens:   estimateTokens(ed),
		})
	}

	used := 0
	var selected []string
	for _, p := range sortContextParts(parts) {
		if used+p.tokens > budget {
			trimmed := trimToTokenBudget(p.text, budget-used)
			if trimmed != "" {
				selected = append(selected, trimmed)
			}
			break
		}
		selected = append(selected, p.text)
		used += p.tokens
	}

	if len(selected) == 0 {
		return ""
	}

	out := strings.Join(selected, "\n\n")
	if b.scanSecrets {
		out = security.RedactSecrets(out)
	}
	return out
}

type contextPart struct {
	priority int
	text     string
	tokens   int
}

func sortContextParts(parts []contextPart) []contextPart {
	out := append([]contextPart(nil), parts...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j].priority < out[i].priority {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func snippetPriority(source string) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "selection":
		return 0
	case "active_file", "editor", "active":
		return 1
	case "mention", "file", "@file":
		return 2
	default:
		return 3
	}
}

func estimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}

	tokens := n / approxCharsPerToken
	if tokens < 1 {
		return 1
	}
	return tokens
}

func trimToTokenBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}
	maxChars := budget * approxCharsPerToken
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	return string(runes[:maxChars]) + "\n...[truncated]"
}

func workspacePrompt(ws *domain.WorkspaceContext) string {
	if ws == nil {
		return ""
	}

	var lines []string
	if n := strings.TrimSpace(ws.Name); n != "" {
		lines = append(lines, "name: "+n)
	}

	if r := strings.TrimSpace(ws.Root); r != "" {
		lines = append(lines, "root: "+r)
	}

	if b := strings.TrimSpace(ws.Branch); b != "" {
		lines = append(lines, "branch: "+b)
	}

	if len(lines) == 0 {
		return ""
	}

	return "Project:\n" + strings.Join(lines, "\n")
}

func selectionPrompt(sel *domain.SelectionContext) string {
	if sel == nil {
		return ""
	}

	text := strings.TrimSpace(sel.Text)
	if text == "" {
		return ""
	}

	header := "Selection"
	if p := strings.TrimSpace(sel.Path); p != "" {
		header += " (" + p
		if l := strings.TrimSpace(sel.Language); l != "" {
			header += ", " + l
		}

		if sel.StartLine != nil && sel.EndLine != nil {
			header += fmt.Sprintf(", lines %d-%d", *sel.StartLine, *sel.EndLine)
		}

		header += ")"
	}

	return header + ":\n" + text
}

func snippetPrompt(sn domain.ContextSnippet) string {
	content := strings.TrimSpace(sn.Content)
	if content == "" {
		return ""
	}

	label := "Snippet"
	if p := strings.TrimSpace(sn.Path); p != "" {
		label = "File: " + p
		if l := strings.TrimSpace(sn.Language); l != "" {
			label += " (" + l + ")"
		}
	}
	if s := strings.TrimSpace(sn.Source); s != "" {
		label += " [" + s + "]"
	}

	return label + ":\n" + content
}

func editorContextPrompt(editor *domain.EditorContext) string {
	if editor == nil {
		return ""
	}

	parts := make([]string, 0, 5)
	if p := strings.TrimSpace(editor.Path); p != "" {
		parts = append(parts, "path: "+p)
	}

	if l := strings.TrimSpace(editor.Language); l != "" {
		parts = append(parts, "language: "+l)
	}

	if editor.CursorLine != nil && editor.CursorColumn != nil {
		parts = append(parts, fmt.Sprintf("cursor: %d:%d", *editor.CursorLine, *editor.CursorColumn))
	}

	if s := strings.TrimSpace(editor.Snippet); s != "" {
		parts = append(parts, "snippet:\n"+s)
	}

	if len(parts) == 0 {
		return ""
	}

	return "Active editor:\n" + strings.Join(parts, "\n")
}

func treePrompt(entries []domain.TreeEntry) string {
	if len(entries) == 0 {
		return ""
	}

	const maxEntries = 200
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}

	var b strings.Builder
	b.WriteString("Project tree:\n")
	for _, e := range entries {
		path := strings.TrimSpace(e.Path)
		if path == "" {
			continue
		}
		kind := strings.TrimSpace(e.Kind)
		if kind == "" {
			kind = "file"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", kind, path)
	}
	if b.Len() <= len("Project tree:\n") {
		return ""
	}
	return b.String()
}
