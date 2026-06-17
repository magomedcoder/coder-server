package contextbudget

import "strings"

const (
	ApproxCharsPerToken = 4
	DefaultTokenBudget  = 8192
	SystemReserveTokens = 512
	MinContextBudget    = 256
)

const (
	PrioritySelection = 0
	PriorityEditor    = 1
	PriorityMention   = 2
	PriorityAuto      = 3
	PriorityRetrieval = 4
	PriorityAux       = 5
	PriorityFolder    = 6
	PriorityUnknown   = 7
	PriorityWorkspace = 10
)

func EffectiveBudget(configured, runnerMax int) int {
	switch {
	case configured > 0 && runnerMax > 0:
		if configured < runnerMax {
			return configured
		}
		return runnerMax
	case configured > 0:
		return configured
	case runnerMax > 0:
		return runnerMax
	default:
		return DefaultTokenBudget
	}
}

func ContextBudgetAfterReserve(totalBudget int, systemTokens int) int {
	if totalBudget <= 0 {
		totalBudget = DefaultTokenBudget
	}

	budget := totalBudget - systemTokens - SystemReserveTokens
	if budget < MinContextBudget {
		return MinContextBudget
	}

	return budget
}

func SnippetPriority(source string) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "selection":
		return PrioritySelection
	case "active_file", "editor", "active":
		return PriorityEditor
	case "mention", "file", "@file":
		return PriorityMention
	case "auto", "import":
		return PriorityAuto
	case "codebase", "search":
		return PriorityRetrieval
	case "terminal", "git", "diagnostics":
		return PriorityAux
	case "folder":
		return PriorityFolder
	default:
		return PriorityUnknown
	}
}

func EstimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}

	tokens := n / ApproxCharsPerToken
	if tokens < 1 {
		return 1
	}

	return tokens
}

func TrimToTokenBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}

	maxChars := budget * ApproxCharsPerToken
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}

	return string(runes[:maxChars]) + "\n...[truncated]"
}
