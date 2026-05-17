package editorprompt

import (
	"fmt"

	"github.com/magomedcoder/gen/api/pb/editorpb"
)

func WrapUserText(text string) string {
	return "Текст:\n\n```\n" + text + "\n```"
}

func BuildSystemPrompt(t editorpb.TransformType, preserveMarkdown bool) string {
	action := "улучши текст"
	switch t {
	case editorpb.TransformType_TRANSFORM_TYPE_FIX:
		action = "исправь орфографию, пунктуацию и грамматику"
	case editorpb.TransformType_TRANSFORM_TYPE_IMPROVE:
		action = "улучши текст: сделай яснее, логичнее и читабельнее, не меняя смысл"
	case editorpb.TransformType_TRANSFORM_TYPE_BEAUTIFY:
		action = "сделай текст более красивым и выразительным, сохраняя смысл"
	case editorpb.TransformType_TRANSFORM_TYPE_PARAPHRASE:
		action = "перефразируй (другими словами), сохраняя смысл"
	case editorpb.TransformType_TRANSFORM_TYPE_SHORTEN:
		action = "сократи текст, сохранив ключевой смысл и факты"
	case editorpb.TransformType_TRANSFORM_TYPE_SIMPLIFY:
		action = "упрости текст: сделай проще и понятнее, без потери смысла"
	case editorpb.TransformType_TRANSFORM_TYPE_MAKE_COMPLEX:
		action = "сделай текст более сложным/профессиональным: добавь точности и терминов, сохраняя смысл"
	case editorpb.TransformType_TRANSFORM_TYPE_MORE_FORMAL:
		action = "перепиши в более формальном стиле"
	case editorpb.TransformType_TRANSFORM_TYPE_MORE_CASUAL:
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
