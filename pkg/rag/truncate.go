package rag

import "unicode/utf8"

func TruncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}

	if utf8.RuneCountInString(s) <= max {
		return s
	}

	r := []rune(s)
	return string(r[:max]) + "\n...(обрезано)"
}
