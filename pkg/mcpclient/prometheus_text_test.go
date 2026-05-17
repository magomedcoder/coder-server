package mcpclient

import (
	"strings"
	"testing"
)

func TestWritePrometheusMetricsStable(t *testing.T) {
	var buf strings.Builder
	if err := WritePrometheusMetrics(&buf); err != nil {
		t.Fatal(err)
	}

	s := buf.String()
	if !strings.Contains(s, "gen_mcp_list_tools_ok") {
		t.Fatalf("missing metric: %s", s)
	}

	if !strings.HasPrefix(s, "# TYPE gen_mcp_") {
		t.Fatalf("unexpected output: %s", s)
	}

	if !strings.Contains(s, "gen_mcp_call_tool_duration_seconds_bucket") {
		t.Fatalf("expected histogram metrics: %s", s)
	}
}

func TestPrometheusMetricNameSanitize(t *testing.T) {
	if prometheusMetricName("call_tool_server_1_ok") != "gen_mcp_call_tool_server_1_ok" {
		t.Fatal()
	}
}
