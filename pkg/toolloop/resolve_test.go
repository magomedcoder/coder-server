package toolloop

import (
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/mcpclient"
)

func TestResolveDeclaredToolNameBareMCPMatchesHexAlias(t *testing.T) {
	orig := "b24_list_tasks"
	alias := mcpclient.ToolAlias(1, orig)
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           alias,
				Description:    "x",
				ParametersJSON: `{}`,
			},
			{
				Name:           "web_search",
				ParametersJSON: `{}`,
			},
		},
	}

	got, ok := ResolveDeclaredToolName(gp, orig)
	if !ok || got != NormalizeToolName(alias) {
		t.Fatalf("resolve bare name: ok=%v got=%q want=%q", ok, got, NormalizeToolName(alias))
	}

	got2, ok2 := ResolveDeclaredToolName(gp, alias)
	if !ok2 || got2 != NormalizeToolName(alias) {
		t.Fatalf("resolve full alias: ok=%v got=%q", ok2, got2)
	}
}

func TestResolveDeclaredToolNameAmbiguousBarePicksLowerServerID(t *testing.T) {
	orig := "ping"
	a1 := mcpclient.ToolAlias(1, orig)
	a2 := mcpclient.ToolAlias(2, orig)
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           a2,
				ParametersJSON: `{}`,
			},
			{
				Name:           a1,
				ParametersJSON: `{}`,
			},
		},
	}

	got, ok := ResolveDeclaredToolName(gp, orig)
	if !ok || got != NormalizeToolName(a1) {
		t.Fatalf("want server 1 first, got ok=%v %q", ok, got)
	}
}

func TestResolveDeclaredToolNameFullAliasKeepsRequestedServer(t *testing.T) {
	orig := "ping"
	a1 := mcpclient.ToolAlias(1, orig)
	a2 := mcpclient.ToolAlias(2, orig)
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           a1,
				ParametersJSON: `{}`,
			},
			{
				Name:           a2,
				ParametersJSON: `{}`,
			},
		},
	}

	got, ok := ResolveDeclaredToolName(gp, a2)
	if !ok || got != NormalizeToolName(a2) {
		t.Fatalf("full alias must keep exact server alias: ok=%v got=%q want=%q", ok, got, NormalizeToolName(a2))
	}
}

func TestResolveDeclaredToolNameUnknown(t *testing.T) {
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           mcpclient.ToolAlias(1, "x"),
				ParametersJSON: `{}`,
			},
		},
	}

	if _, ok := ResolveDeclaredToolName(gp, "nonexistent_tool_xyz"); ok {
		t.Fatal("expected no match")
	}
}

func TestResolveDeclaredToolNameRecoversWrongHexWhenSingleToolOnServer(t *testing.T) {
	orig := "ping"
	canon := mcpclient.ToolAlias(1, orig)
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           canon,
				ParametersJSON: `{}`,
			},
		},
	}
	hallucinated := "mcp_1_h1234567890abcdef"
	got, ok := ResolveDeclaredToolName(gp, hallucinated)
	if !ok || got != NormalizeToolName(canon) {
		t.Fatalf("recover single MCP tool on server: ok=%v got=%q want=%q", ok, got, NormalizeToolName(canon))
	}
}

func TestResolveDeclaredToolNameNoRecoverWhenTwoToolsOnSameServer(t *testing.T) {
	a1 := mcpclient.ToolAlias(1, "ping")
	a2 := mcpclient.ToolAlias(1, "pong")
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           a1,
				ParametersJSON: `{}`,
			},
			{
				Name:           a2,
				ParametersJSON: `{}`,
			},
		},
	}
	if _, ok := ResolveDeclaredToolName(gp, "mcp_1_h1234567890abcdef"); ok {
		t.Fatal("ambiguous server_id=1: must not guess tool from fake hex")
	}
}
