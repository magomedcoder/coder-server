package security

import (
	"regexp"
	"strings"
)

var jailbreakPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore (all )?(previous|prior|above) instructions`),
	regexp.MustCompile(`(?i)disregard (your|the) (system|initial) prompt`),
	regexp.MustCompile(`(?i)you are now (in )?dan\b`),
	regexp.MustCompile(`(?i)jailbreak`),
	regexp.MustCompile(`(?i)reveal (your|the) system prompt`),
}

func DetectPromptInjection(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	for _, re := range jailbreakPatterns {
		if re.MatchString(text) {
			return true
		}
	}

	return false
}

func ScanMessages(messages []string) bool {
	for _, m := range messages {
		if DetectPromptInjection(m) {
			return true
		}
	}

	return false
}
