package mcpclient

const (
	MaxMetaListItems      = 2000
	MaxMetaToolReplyRunes = 120_000
)

func TruncateLLMReply(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return s
	}

	var n int
	for i := range s {
		if n >= maxRunes {
			return s[:i] + "\n\n[GEN: ответ обрезан по лимиту контекста]"
		}
		n++
	}

	return s
}
