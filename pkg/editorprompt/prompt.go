package editorprompt

import "fmt"

type TransformType int

const (
	TransformUnspecified TransformType = iota
	TransformFix
	TransformImprove
	TransformBeautify
	TransformParaphrase
	TransformShorten
	TransformSimplify
	TransformMakeComplex
	TransformMoreFormal
	TransformMoreCasual
)

func WrapUserText(text string) string {
	return "Текст:\n\n```\n" + text + "\n```"
}

func BuildSystemPrompt(t TransformType, preserveMarkdown bool) string {
	action := "улучши текст"
	switch t {
	case TransformFix:
		action = "исправь орфографию, пунктуацию и грамматику"
	case TransformImprove:
		action = "улучши текст: сделай яснее, логичнее и читабельнее, не меняя смысл"
	case TransformBeautify:
		action = "сделай текст более красивым и выразительным, сохраняя смысл"
	case TransformParaphrase:
		action = "перефразируй (другими словами), сохраняя смысл"
	case TransformShorten:
		action = "сократи текст, сохранив ключевой смысл и факты"
	case TransformSimplify:
		action = "упрости текст: сделай проще и понятнее, без потери смысла"
	case TransformMakeComplex:
		action = "сделай текст более сложным/профессиональным: добавь точности и терминов, сохраняя смысл"
	case TransformMoreFormal:
		action = "перепиши в более формальном стиле"
	case TransformMoreCasual:
		action = "перепиши в разговорном стиле"
	default:
		action = "улучши текст"
	}

	formatRule := "Сохраняй переносы строк и структуру по смыслу."
	if preserveMarkdown {
		formatRule = "Сохраняй Markdown/разметку, списки и переносы строк (если они есть)."
	}

	return fmt.Sprintf(
		"Ты - редактор текста. Задача: %s.\n"+
			"Правила:\n"+
			"- Верни ТОЛЬКО итоговый отредактированный текст, без пояснений.\n"+
			"- Язык результата: тот же, что и исходный текст; если язык неочевиден, используй русский.\n"+
			"- Сохраняй смысл; не добавляй новых фактов.\n"+
			"- Имена, числа, даты и сущности не меняй (кроме явных опечаток).\n"+
			"- %s\n",
		action, formatRule,
	)
}
