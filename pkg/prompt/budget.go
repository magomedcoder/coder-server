package prompt

import (
	"strings"
	"unicode/utf8"

	"github.com/magomedcoder/coder-server/pkg/domain"
	"github.com/magomedcoder/coder-server/pkg/llmhistory"
)

type InstructionSafeBudgetMetrics struct {
	DroppedRunesTotal    int
	DroppedRunesByFile   map[string]int
	DroppedRunesBySource map[string]int
}

func (m *InstructionSafeBudgetMetrics) AddDrop(block DocumentContextBlock, droppedRunes int) {
	if droppedRunes <= 0 {
		return
	}

	m.DroppedRunesTotal += droppedRunes
	if m.DroppedRunesByFile == nil {
		m.DroppedRunesByFile = make(map[string]int)
	}

	if m.DroppedRunesBySource == nil {
		m.DroppedRunesBySource = make(map[string]int)
	}

	file := strings.TrimSpace(block.SourceFile)
	if file == "" {
		file = strings.TrimSpace(block.Title)
	}

	src := strings.TrimSpace(block.SourceType)
	if src == "" {
		src = "unknown"
	}

	m.DroppedRunesByFile[file] += droppedRunes
	m.DroppedRunesBySource[src] += droppedRunes
}

func TruncateDocumentContextBody(body string, maxRunes int) (string, int) {
	if maxRunes <= 0 {
		return "", utf8.RuneCountInString(body)
	}

	r := []rune(body)
	if len(r) <= maxRunes {
		return body, 0
	}

	notice := "\n\n...(обрезано instruction-safe budget manager)"
	noticeRunes := utf8.RuneCountInString(notice)
	keep := max(maxRunes-noticeRunes, 0)
	truncated := string(r[:keep]) + notice
	return truncated, len(r) - keep
}

func ApplyInstructionSafeBudgetManager(maxContextTokens int, systemPolicy *domain.Message, history []*domain.Message, userInstruction *domain.Message, blocks []DocumentContextBlock) ([]DocumentContextBlock, InstructionSafeBudgetMetrics) {
	metrics := InstructionSafeBudgetMetrics{}
	if len(blocks) == 0 {
		return blocks, metrics
	}

	if maxContextTokens <= 0 {
		return blocks, metrics
	}

	baseMessages := AssemblePromptMessages(0, systemPolicy, history, userInstruction, nil)
	baseTok := llmhistory.SumApproxTokens(baseMessages)
	const preserveOutputTokens = 512
	availableDocTok := maxContextTokens - baseTok - preserveOutputTokens
	if availableDocTok <= 0 {
		for _, blk := range blocks {
			metrics.AddDrop(blk, utf8.RuneCountInString(blk.Body))
		}
		return nil, metrics
	}

	availableRunes := max(availableDocTok*2-utf8.RuneCountInString(DocumentContextHierarchyInstruction), 0)
	if availableRunes <= 0 {
		for _, blk := range blocks {
			metrics.AddDrop(blk, utf8.RuneCountInString(blk.Body))
		}
		return nil, metrics
	}

	out := make([]DocumentContextBlock, 0, len(blocks))
	usedRunes := 0
	for _, blk := range blocks {
		title := strings.TrimSpace(blk.Title)
		body := strings.TrimSpace(blk.Body)
		if body == "" {
			continue
		}

		formatted := FormatDocumentContextBlock(title, body)
		blockRunes := utf8.RuneCountInString(formatted)
		if usedRunes+blockRunes <= availableRunes {
			out = append(out, blk)
			usedRunes += blockRunes
			continue
		}

		room := availableRunes - usedRunes
		if room < 180 {
			metrics.AddDrop(blk, utf8.RuneCountInString(body))
			continue
		}

		headingRunes := utf8.RuneCountInString(FormatDocumentContextBlock(title, ""))
		bodyBudget := room - headingRunes
		if bodyBudget < 80 {
			metrics.AddDrop(blk, utf8.RuneCountInString(body))
			continue
		}

		newBody, dropped := TruncateDocumentContextBody(body, bodyBudget)
		if strings.TrimSpace(newBody) == "" {
			metrics.AddDrop(blk, utf8.RuneCountInString(body))
			continue
		}

		blk.Body = newBody
		out = append(out, blk)
		usedRunes += utf8.RuneCountInString(FormatDocumentContextBlock(title, newBody))
		metrics.AddDrop(blk, dropped)
	}

	return out, metrics
}
