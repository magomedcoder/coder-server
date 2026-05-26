package service

import (
	"regexp"
	"strings"
)

var (
	rustErrorLine = regexp.MustCompile(`(?m)^error(\[[\w-]+\])?: .+`)
	rustPanicLine = regexp.MustCompile(`(?m)^panicked at .+|^thread '.+' panicked at .+`)
	pyTraceLine   = regexp.MustCompile(`(?m)^(?:\w+Error|SyntaxError|IndentationError|AssertionError): .+`)
	pyFileLine    = regexp.MustCompile(`(?m)^  File ".+", line \d+`)
	tsErrorLine   = regexp.MustCompile(`(?m)^.*(?:TS\d+|SyntaxError|TypeError|ReferenceError): .+`)
	tsStackLine   = regexp.MustCompile(`(?m)^\s+at .+ \(.+:\d+:\d+\)`)
)

func ParseTraceback(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}

	for _, re := range []*regexp.Regexp{rustErrorLine, rustPanicLine, pyTraceLine, tsErrorLine} {
		if m := re.FindString(stderr); m != "" {
			return truncate(m, 300)
		}
	}

	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if pyFileLine.MatchString(line) || tsStackLine.MatchString(line) {
			return truncate(line, 300)
		}
	}

	return extractTracebackLine(stderr)
}
