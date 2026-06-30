package rag

import (
	"strings"
	"unicode/utf8"
)

const QueryRewriteSystem = `Ты превращаешь вопрос пользователя в краткий поисковый запрос для семантического поиска по фрагментам документов.

Правила:
- Сохраняй язык сообщения пользователя (в том числе при многоязычном вводе).
- Выведи только текст поискового запроса, одна строка (без кавычек, подписей, ограждений кода).
- Сохраняй ключевые сущности, числа и имена собственные.
- Если ввод уже оптимален, повтори его с минимальными правками.`

const QueryRewriteMaxRunes = 512
const HyDEMaxRunes = 1200

const HyDESystem = `Ты пишешь компактный гипотетический отрывок, в котором с высокой вероятностью содержится ответ на вопрос пользователя.

Правила:
- Тот же язык, что и у пользователя.
- Только простой текст (без markdown, маркированных списков, вступлений).
- Включи важные сущности, термины и ограничения из вопроса.
- 3–8 предложений, фактический стиль, без оговорок.`

func SanitizeRewrittenQuery(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		first := strings.TrimSpace(s[:idx])
		if first != "" {
			s = first
		}
	}

	return capQueryRunes(s, QueryRewriteMaxRunes)
}

func SanitizeHyDEPseudoDocument(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	return capQueryRunes(s, HyDEMaxRunes)
}

func capQueryRunes(s string, max int) string {
	if max <= 0 {
		return s
	}

	if utf8.RuneCountInString(s) <= max {
		return s
	}

	r := []rune(s)
	return string(r[:max])
}
