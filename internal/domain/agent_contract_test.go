package domain

import (
	"encoding/json"
	"testing"
)

const agentStepRequestFixture = `{
  "request_id": "agent-session-1-step-0",
  "session_id": "session-1",
  "goal": "Fix tests in parser",
  "observations": [
    {
      "call_id": "call-1",
      "tool": "read_file",
      "ok": true,
      "result": {
		"path": "src/main.rs",
		"content": "fn main() {}"
      }
    }
  ]
}`

const agentStepResponseFixture = `{
  "finish": false,
  "summary": "Reading source files",
  "calls": [
    {
		"tool": "read_file",
		"id": "call-1",
		"args": {
		    "path": "src/main.rs"
		}
    }
  ],
  "step": 1,
  "blocked": []
}`

func TestAgentStepRequestFixture(t *testing.T) {
	var req AgentStepRequest
	if err := json.Unmarshal([]byte(agentStepRequestFixture), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if req.RequestID != "agent-session-1-step-0" {
		t.Fatalf("request_id=%q", req.RequestID)
	}

	if req.SessionID != "session-1" {
		t.Fatalf("session_id=%q", req.SessionID)
	}

	if len(req.Observations) != 1 {
		t.Fatalf("observations=%d", len(req.Observations))
	}
}

func TestAgentStepResponseFixture(t *testing.T) {
	var resp AgentStepResponse
	if err := json.Unmarshal([]byte(agentStepResponseFixture), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Finish {
		t.Fatal("finish должно быть false")
	}

	if len(resp.Calls) != 1 || resp.Calls[0].Tool != "read_file" {
		t.Fatalf("calls=%+v", resp.Calls)
	}

	if resp.Step != 1 {
		t.Fatalf("step=%d", resp.Step)
	}
}
