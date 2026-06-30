package mcpclient

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

var (
	callToolDurationSumNs atomic.Uint64
	callToolDurationCount atomic.Uint64
	callToolHistBucket    [4]atomic.Uint64
)

func recordCallToolDuration(d time.Duration) {
	if d < 0 {
		d = 0
	}
	sec := d.Seconds()
	callToolDurationSumNs.Add(uint64(d.Nanoseconds()))
	callToolDurationCount.Add(1)
	limits := []float64{0.05, 0.2, 1, 5}
	for i := range limits {
		if sec <= limits[i] {
			callToolHistBucket[i].Add(1)
		}
	}
}

func writeCallToolDurationHistogram(w io.Writer) {
	fmt.Fprintf(w, "# HELP gen_mcp_call_tool_duration_seconds Длительность MCP tools/call (сек)\n")
	fmt.Fprintf(w, "# TYPE gen_mcp_call_tool_duration_seconds histogram\n")
	le := []string{"0.05", "0.2", "1", "5"}
	for i := range callToolHistBucket {
		fmt.Fprintf(w, "gen_mcp_call_tool_duration_seconds_bucket{le=%q} %d\n", le[i], callToolHistBucket[i].Load())
	}

	cnt := callToolDurationCount.Load()
	fmt.Fprintf(w, "gen_mcp_call_tool_duration_seconds_bucket{le=\"+Inf\"} %d\n", cnt)
	sumSec := float64(callToolDurationSumNs.Load()) / 1e9
	fmt.Fprintf(w, "gen_mcp_call_tool_duration_seconds_sum %g\n", sumSec)
	fmt.Fprintf(w, "gen_mcp_call_tool_duration_seconds_count %d\n", cnt)
}
