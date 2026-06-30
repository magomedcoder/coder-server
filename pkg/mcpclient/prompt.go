package mcpclient

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/magomedcoder/lmpkg/domain"
)

const MaxToolNamesListRunes = 12000

type ServerEntry struct {
	ID      int64
	Name    string
	Enabled bool
}

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

	return truncatePromptRunes(text, MaxToolNamesListRunes, func(total int) string {
		return fmt.Sprintf("\n...(обрезано, всего инструментов=%d)", total)
	}, len(tools))
}

func TruncateCommaList(list string, totalNames int) string {
	return truncatePromptRunes(list, MaxToolNamesListRunes, func(total int) string {
		return fmt.Sprintf("...(обрезано, всего=%d)", total)
	}, totalNames)
}

func FormatServerLine(e ServerEntry) string {
	if e.ID <= 0 {
		return ""
	}

	line := fmt.Sprintf("- id=%d", e.ID)
	if !e.Enabled {
		return fmt.Sprintf("- id=%d  (отключён в каталоге)", e.ID)
	}

	if n := strings.TrimSpace(e.Name); n != "" {
		line = fmt.Sprintf("- id=%d  %s", e.ID, n)
	}

	return line
}

func BuildSessionHints(entries []ServerEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("[MCP] В этой сессии чата включены внешние инструменты. Разрешённые server_id (используй только их):\n")
	for _, e := range entries {
		if line := FormatServerLine(e); line != "" {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	b.WriteString("\nИмена MCP-инструментов вида mcp_<id>_h<hex> задаёт платформа; hex привязан к реальному имени на сервере. Точные имена для этого запроса - в блоке [Tools] ниже (если он есть).\n")
	b.WriteString("КРИТИЧНО - имена tools:\n")
	b.WriteString("- Копируй поле name из блока [Tools] / списка tools СИМВОЛ В СИМВОЛ. Не сокращай, не «улучшай», не подставляй примеры вроде mcp_1_h123456 или шаблонные hex.\n")
	b.WriteString("- Любое другое имя (включая похожее на mcp_<id>_h...) не будет выполнено: платформа не угадает вашу замену.\n")
	b.WriteString("КРИТИЧНО - как выполняется вызов:\n")
	b.WriteString("- Недостаточно описать вызов в свободном тексте («предположу инструмент...», «если вернётся...»). Чтобы инструмент реально вызвался, в ответе должен быть машиночитаемый вызов в формате, ожидаемом для tool-calling (JSON-массив с полями tool_name и parameters и/или блок ```json ... ``` - как в вашей инструкции к модели).\n")
	b.WriteString("- Сначала вызови релевантный tool с корректными аргументами по его JSON-схеме, получи данные, затем формируй ответ пользователю по факту результата.\n")
	b.WriteString("Не добавляй в аргументы поле server_id: привязка к серверу уже зашита в имени инструмента.\n")
	b.WriteString("Не утверждай, что инструмента нет или что доступ невозможен, пока не проверишь доступные tools и не выполнил релевантный вызов.\n")

	return strings.TrimSpace(b.String())
}

func AppendBlock(content, block string) string {
	block = strings.TrimSpace(block)
	if block == "" {
		return content
	}

	if strings.TrimSpace(content) == "" {
		return block
	}

	return content + "\n\n" + block
}

func truncatePromptRunes(text string, maxRunes int, suffix func(total int) string, total int) string {
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}

	runes := []rune(text)
	return string(runes[:maxRunes]) + suffix(total)
}
