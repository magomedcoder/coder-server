package mcpprompt

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/magomedcoder/gen/pkg/domain"
)

const MaxToolNamesListRunes = 12000

func BuildToolCatalog(tools []domain.Tool) string {
	if len(tools) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("[Tools] Разрешённые инструменты в этом запросе - в вызовах используй только эти значения name (символ в символ):\n")
	for _, t := range tools {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			name = "(без имени)"
		}

		b.WriteString("- ")
		b.WriteString(name)
		b.WriteByte('\n')
	}

	text := strings.TrimSpace(b.String())
	if text == "" {
		return ""
	}

	return truncateRunes(text, MaxToolNamesListRunes, func(total int) string {
		return fmt.Sprintf("\n…(обрезано, всего инструментов=%d)", total)
	}, len(tools))
}

func TruncateCommaList(list string, totalNames int) string {
	return truncateRunes(list, MaxToolNamesListRunes, func(total int) string {
		return fmt.Sprintf("…(обрезано, всего=%d)", total)
	}, totalNames)
}

func truncateRunes(text string, maxRunes int, suffix func(total int) string, total int) string {
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}

	runes := []rune(text)
	return string(runes[:maxRunes]) + suffix(total)
}
