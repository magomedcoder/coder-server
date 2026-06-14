package security

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|passwd)\s*[:=]\s*\S+`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`ghp_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]+ PRIVATE KEY-----`),
}

const redacted = "[REDACTED]"

func RedactSecrets(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	out := text
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, redacted)
	}

	return out
}

func ContainsSecrets(text string) bool {
	for _, re := range secretPatterns {
		if re.MatchString(text) {
			return true
		}
	}

	return false
}
