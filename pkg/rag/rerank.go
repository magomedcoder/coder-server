package rag

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const RerankSystem = `Ты ранжируешь текстовые фрагменты по релевантности к вопросу пользователя.

Правила:
- Фрагменты пронумерованы 1..N в сообщении пользователя в заданном порядке.
- Выведи ТОЛЬКО числа от 1 до N через запятую (сначала лучший, в конце худший).
- Каждое число от 1 до N должно встретиться ровно один раз.
- Без пояснений, лишних слов и markdown.`

var rerankDigitSplit = regexp.MustCompile(`[^0-9]+`)

func TrimPassageForRerank(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || s == "" {
		return s
	}

	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}

	r := []rune(s)
	return string(r[:maxRunes]) + "..."
}

func ParseRerankOrder(reply string, n int) []int {
	if n <= 0 {
		return nil
	}
	reply = strings.TrimSpace(reply)
	reply = strings.TrimPrefix(reply, "```")
	reply = strings.TrimSuffix(reply, "```")
	reply = strings.TrimSpace(reply)

	var raw []string
	if strings.Contains(reply, ",") {
		for p := range strings.SplitSeq(reply, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}

			raw = append(raw, p)
		}
	} else {
		for _, tok := range rerankDigitSplit.Split(reply, -1) {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}

			raw = append(raw, tok)
		}
	}

	seen := make(map[int]struct{})
	var order []int
	for _, p := range raw {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 || v > n {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		order = append(order, v-1)
	}

	if len(order) < n {
		for i := range n {
			if _, ok := seen[i+1]; !ok {
				order = append(order, i)
			}
		}
	}

	if len(order) > n {
		order = order[:n]
	}

	return order
}
