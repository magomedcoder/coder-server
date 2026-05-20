package runnerprompt

import (
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/mcpprompt"
)

func BuildToolsInvocationBlock(tools []domain.Tool) string {
	if len(tools) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\nTools:\n")
	for _, t := range tools {
		b.WriteString("- ")
		b.WriteString(strings.TrimSpace(t.Name))
		if d := strings.TrimSpace(t.Description); d != "" {
			b.WriteString(": ")
			b.WriteString(d)
		}

		if p := strings.TrimSpace(t.ParametersJSON); p != "" {
			b.WriteString(" (params: ")
			b.WriteString(p)
			b.WriteString(")")
		}
		b.WriteByte('\n')
	}
	b.WriteString("\nЧтобы вызвать инструмент, верни один JSON-массив (можно в блоке ```json), строго в формате:\n")
	b.WriteString(`[{"tool_name":"<имя из списка>","parameters":{...}}]`)
	b.WriteString("\n\nПоле parameters - объект JSON; если параметров нет, используй {}.\n")

	return strings.TrimSpace(b.String())
}

func AppendToolsInvocationToSystem(msg *domain.Message, tools []domain.Tool) {
	if msg == nil || len(tools) == 0 {
		return
	}

	if block := BuildToolsInvocationBlock(tools); block != "" {
		msg.Content = mcpprompt.AppendBlock(msg.Content, block)
	}
}
