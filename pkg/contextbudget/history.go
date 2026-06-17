package contextbudget

import "strings"

const (
	perMessageOverheadTokens = 4
	historySafetyMargin      = 256
)

// HistoryTokenBudget возвращает число токенов, оставшихся для сообщений чата после фиксированных блоков промпта
func HistoryTokenBudget(totalBudget, systemTokens, contextTokens, generateMaxTokens int) int {
	if totalBudget <= 0 {
		totalBudget = DefaultTokenBudget
	}

	reserved := systemTokens + contextTokens + generateMaxTokens + SystemReserveTokens + historySafetyMargin
	budget := totalBudget - reserved
	if budget < MinContextBudget {
		return MinContextBudget
	}

	return budget
}

type ChatLine struct {
	Role    string
	Content string
}

func EstimateChatLineTokens(line ChatLine) int {
	content := strings.TrimSpace(line.Content)
	if content == "" {
		return 0
	}

	return EstimateTokens(content) + perMessageOverheadTokens
}

// TrimChatLines удаляет самые старые сообщения; последнее сообщение сохраняется, если оно есть
func TrimChatLines(lines []ChatLine, maxTokens int) ([]ChatLine, bool) {
	if maxTokens <= 0 || len(lines) == 0 {
		return lines, false
	}

	total := 0
	for _, line := range lines {
		total += EstimateChatLineTokens(line)
	}

	if total <= maxTokens {
		return lines, false
	}

	trimmed := make([]ChatLine, len(lines))
	copy(trimmed, lines)
	changed := false
	for len(trimmed) > 1 && sumChatLineTokens(trimmed) > maxTokens {
		trimmed = trimmed[1:]
		changed = true
	}

	if len(trimmed) == 1 && sumChatLineTokens(trimmed) > maxTokens {
		line := trimmed[0]
		line.Content = TrimToTokenBudget(line.Content, maxTokens-perMessageOverheadTokens)
		trimmed[0] = line
		changed = true
	}

	return trimmed, changed
}

func sumChatLineTokens(lines []ChatLine) int {
	total := 0
	for _, line := range lines {
		total += EstimateChatLineTokens(line)
	}

	return total
}
