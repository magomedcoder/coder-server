package prompt

import (
	"encoding/json"
	"strings"

	"github.com/magomedcoder/lm-runner/template"
)

func parseToolCallsJSON(raw string) ([]template.ToolCall, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var rawCalls []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal([]byte(raw), &rawCalls); err != nil {
		return nil, err
	}

	out := make([]template.ToolCall, 0, len(rawCalls))
	for i, rc := range rawCalls {
		tc := template.ToolCall{
			ID: rc.ID,
			Function: template.ToolCallFunction{
				Index: i,
				Name:  strings.TrimSpace(rc.Function.Name),
			},
		}

		args := rc.Function.Arguments
		if len(args) == 0 || string(args) == "null" {
			tc.Function.Arguments.M = map[string]any{}
			out = append(out, tc)
			continue
		}

		var argBytes []byte
		if args[0] == '"' {
			var s string
			if err := json.Unmarshal(args, &s); err != nil {
				tc.Function.Arguments.M = map[string]any{}
				out = append(out, tc)
				continue
			}

			argBytes = []byte(strings.TrimSpace(s))
		} else {
			argBytes = args
		}

		if len(argBytes) == 0 {
			tc.Function.Arguments.M = map[string]any{}
		} else if err := json.Unmarshal(argBytes, &tc.Function.Arguments); err != nil {
			tc.Function.Arguments.M = map[string]any{"_raw": string(argBytes)}
		}

		if tc.Function.Arguments.M == nil {
			tc.Function.Arguments.M = map[string]any{}
		}

		out = append(out, tc)
	}

	return out, nil
}
