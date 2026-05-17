package toolloop

import (
	"strings"
)

func ErrorToolMessage(call ExecutableToolCall, err error, partialResult string, deadlineExceeded bool) string {
	var b strings.Builder
	b.WriteString("Статус: error выполнения инструмента.\n")
	b.WriteString("Твоя задача: кратко и по-русски объясни пользователю, что пошло не так и что можно сделать (повторить request, проверить права, уточнить параметры).\n\n")
	if deadlineExceeded {
		b.WriteString("Причина: истёк timeout ожидания responseа инструмента.\n")
	} else if err != nil {
		b.WriteString("Причина (технически): ")
		b.WriteString(strings.TrimSpace(err.Error()))
		b.WriteByte('\n')
	}

	b.WriteString("Запрошенное имя инструмента: ")
	b.WriteString(strings.TrimSpace(call.RequestedName))
	b.WriteString("\nВнутреннее имя: ")
	b.WriteString(strings.TrimSpace(call.ResolvedName))
	b.WriteByte('\n')
	pr := strings.TrimSpace(partialResult)
	errText := ""
	if err != nil {
		errText = strings.TrimSpace(err.Error())
	}

	if pr != "" && pr != errText {
		b.WriteString("\nДополнительно от сервера или среды выполнения:\n")
		b.WriteString(pr)
		b.WriteByte('\n')
	}

	return TruncateToolResult(strings.TrimSpace(b.String()))
}
