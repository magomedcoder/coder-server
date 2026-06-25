package llmhistory

import (
	"strings"
	"unicode/utf8"

	"github.com/magomedcoder/lmpkg/domain"
)

func EffectiveMaxRAGContextRunes(maxContextTokens int, runesCeiling int, systemAndHistory []*domain.Message, userMessage string) int {
	if runesCeiling <= 0 {
		runesCeiling = 32000
	}
	if maxContextTokens <= 0 {
		const blindRAGRunes = 3200
		if blindRAGRunes < runesCeiling {
			return blindRAGRunes
		}
		return runesCeiling
	}

	pre := SumApproxTokens(systemAndHistory)
	const genReserve = 512
	userOverhead := max(utf8.RuneCountInString(strings.TrimSpace(userMessage))/2, 32)

	ragTok := max(maxContextTokens-pre-genReserve-userOverhead, 120)
	runesLimit := max(min(ragTok*2, runesCeiling), 200)
	return runesLimit
}
