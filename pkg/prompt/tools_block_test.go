package prompt

import (
	"strings"
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
)

func TestBuildToolsInvocationBlock_containsFormatHint(t *testing.T) {
	block := BuildToolsInvocationBlock([]domain.Tool{{
		Name:           "web_search",
		Description:    "search",
		ParametersJSON: `{"type":"object"}`,
	}})
	if !strings.Contains(block, "web_search") || !strings.Contains(block, "tool_name") {
		t.Fatalf("неожиданный блок: %q", block)
	}
}
