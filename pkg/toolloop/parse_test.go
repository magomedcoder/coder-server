package toolloop

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/magomedcoder/coder-server/pkg/chatstream"
	"github.com/magomedcoder/coder-server/pkg/domain"
)

func TestExtractLeadingJSONArray(t *testing.T) {
	raw := `[{"tool_name":"build_docx","parameters":{"spec_json":"{}"}}]`
	if got := ExtractLeadingJSONArray("  \n" + raw + "\n"); got != raw {
		t.Fatalf("ExtractLeadingJSONArray: %q", got)
	}
	withNoise := `[{"note":"edge: ] and [ in string","tool_name":"x","parameters":{}}]`
	if got := ExtractLeadingJSONArray(withNoise); got != withNoise {
		t.Fatalf("внутри строки скобки: %q", got)
	}
}

func TestExtractToolActionBlob_rawPrefix(t *testing.T) {
	blob := ExtractToolActionBlob(`[{"tool_name":"apply_spreadsheet","parameters":{"operations_json":"[]"}}]`)
	rows, err := ParseCohereActionList(blob)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "apply_spreadsheet" {
		t.Fatalf("blob=%q err=%v rows=%v", blob, err, rows)
	}
}

func TestExtractToolActionBlob_embeddedAfterPreamble(t *testing.T) {
	text := "Кратко: обновлю книгу.\n\n" +
		`[{"tool_name":"apply_spreadsheet","parameters":{"operations_json":"[]"}}]` +
		"\n\nГотово."
	blob := ExtractToolActionBlob(text)
	rows, err := ParseCohereActionList(blob)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "apply_spreadsheet" {
		t.Fatalf("blob=%q err=%v rows=%v", blob, err, rows)
	}
}

func TestExtractToolActionBlob_genericCodeFence(t *testing.T) {
	text := "Вот вызов:\n\n```\n" +
		`[{"tool_name":"build_docx","parameters":{"spec_json":"{}"}}]` +
		"\n```\n"
	blob := ExtractToolActionBlob(text)
	rows, err := ParseCohereActionList(blob)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "build_docx" {
		t.Fatalf("blob=%q err=%v rows=%v", blob, err, rows)
	}
}

func TestExtractCohereActionJSON(t *testing.T) {
	text := `Краткое рассуждение здесь.

Действие: ` + "```json\n[\n  {\"tool_name\": \"apply_spreadsheet\", \"parameters\": {\"operations_json\": \"[]\"}}\n]\n```"

	got := ExtractCohereActionJSON(text)
	if !strings.Contains(got, "apply_spreadsheet") {
		t.Fatalf("ожидался JSON с apply_spreadsheet, получено: %q", got)
	}
	rows, err := ParseCohereActionList(got)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ToolName != "apply_spreadsheet" {
		t.Fatalf("неверный разбор: %+v", rows)
	}
}

func TestParseCohereActionList_legacyNameArguments(t *testing.T) {
	legacy := `{"name":"apply_spreadsheet","arguments":{"operations_json":"[]"}}`
	rows, err := ParseCohereActionList(legacy)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "apply_spreadsheet" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestParseCohereActionList_legacyToolArguments(t *testing.T) {
	legacy := `{"tool":"b24_get_task","arguments":{"task_id":1001}}`
	rows, err := ParseCohereActionList(legacy)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "b24_get_task" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}

	if string(rows[0].Parameters) != `{"task_id":1001}` {
		t.Fatalf("неожиданные arguments: %s", string(rows[0].Parameters))
	}
}

func TestParseCohereActionList_StringifiedArgumentsJSON(t *testing.T) {
	legacy := `{"tool":"b24_get_task","arguments":"{\"task_id\":1001}"}`
	rows, err := ParseCohereActionList(legacy)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "b24_get_task" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}

	if string(rows[0].Parameters) != `{"task_id":1001}` {
		t.Fatalf("неожиданные normalized arguments: %s", string(rows[0].Parameters))
	}
}

func TestExtractToolActionBlob_leadingLegacyObject(t *testing.T) {
	text := `{"name":"build_docx","arguments":{"spec_json":"{}"}}`
	blob := ExtractToolActionBlob(text)
	rows, err := ParseCohereActionList(blob)
	if err != nil || len(rows) != 1 || rows[0].ToolName != "build_docx" {
		t.Fatalf("blob=%q err=%v rows=%+v", blob, err, rows)
	}
}

func TestFilterExecutableToolRows(t *testing.T) {
	rows := []CohereActionRow{
		{ToolName: "directly-answer", Parameters: []byte(`{"answer":"привет"}`)},
		{ToolName: "apply_spreadsheet", Parameters: []byte(`{"operations_json":"[]"}`)},
	}
	out := FilterExecutableToolRows(rows)
	if len(out) != 1 || out[0].ToolName != "apply_spreadsheet" {
		t.Fatalf("ожидалась одна строка apply_spreadsheet, получено %+v", out)
	}
}

func TestToolExecutionDuration(t *testing.T) {
	if d := ToolExecutionDuration(0); d != DefaultToolExecSeconds*time.Second {
		t.Fatalf("0 -> %v, ожидалось %v", d, DefaultToolExecSeconds*time.Second)
	}
	if d := ToolExecutionDuration(10); d != MinToolExecSeconds*time.Second {
		t.Fatalf("10 -> %v, ожидалось %v", d, MinToolExecSeconds*time.Second)
	}
	if d := ToolExecutionDuration(600); d != MaxToolExecSeconds*time.Second {
		t.Fatalf("600 -> %v, ожидалось %v", d, MaxToolExecSeconds*time.Second)
	}
	if d := ToolExecutionDuration(90); d != 90*time.Second {
		t.Fatalf("90 -> %v", d)
	}
}

func TestRunFnWithContextNoDeadline(t *testing.T) {
	v, err := RunFnWithContext(context.Background(), func() (int, error) {
		return 42, nil
	})
	if err != nil || v != 42 {
		t.Fatalf("получено %v, %v", v, err)
	}
}

func TestDrainLLMStreamChannelForward(t *testing.T) {
	ch := make(chan domain.LLMStreamChunk, 2)
	go func() {
		ch <- domain.LLMStreamChunk{Content: "a"}
		ch <- domain.LLMStreamChunk{Content: "b"}
		close(ch)
	}()

	var got []domain.LLMStreamChunk
	raw, streamed := DrainLLMStreamChannelForward(ch, func(c domain.LLMStreamChunk) bool {
		got = append(got, c)
		return true
	})

	if raw != "ab" || !streamed || len(got) != 2 || got[0].Content != "a" || got[1].Content != "b" {
		t.Fatalf("raw=%q streamed=%v получено=%v", raw, streamed, got)
	}
}

func TestStreamToolRoundComplete(t *testing.T) {
	var chunks []chatstream.ChatStreamChunk
	send := func(c chatstream.ChatStreamChunk) bool {
		chunks = append(chunks, c)
		return true
	}

	StreamToolRoundComplete(send, 7, false, "x", "x")

	if len(chunks) != 1 || chunks[0].MessageID != 7 || chunks[0].Text != "x" {
		t.Fatalf("нет стрима: %+v", chunks)
	}

	chunks = nil
	StreamToolRoundComplete(send, 8, true, "same", "same")
	if len(chunks) != 1 || chunks[0].MessageID != 8 || chunks[0].Text != "" {
		t.Fatalf("стрим same: %+v", chunks)
	}

	chunks = nil
	StreamToolRoundComplete(send, 9, true, "raw", "short")
	if len(chunks) != 1 || chunks[0].MessageID != 9 || chunks[0].Text != "short" {
		t.Fatalf("стрим diff: %+v", chunks)
	}
}

func TestMaxToolInvocationRoundsDefaultsAndClamp(t *testing.T) {
	if got := MaxToolInvocationRounds(nil); got != DefaultToolLoopRounds {
		t.Fatalf("по умолчанию rounds несовпадение: получено=%d ожидалось=%d", got, DefaultToolLoopRounds)
	}
}

func TestResolveExecutableToolCallsUsesResolvedAliases(t *testing.T) {
	alias := "mcp_9_h70696e67"
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{Name: alias},
		},
	}

	rows := []CohereActionRow{
		{
			ToolName:   "ping",
			Parameters: []byte(`{"x":1}`),
		},
	}

	out, err := ResolveExecutableToolCalls(gp, rows)
	if err != nil {
		t.Fatalf("неожиданась ошибка: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("неожиданное calls count: %d", len(out))
	}

	if out[0].RequestedName != "ping" {
		t.Fatalf("requested name несовпадение: %q", out[0].RequestedName)
	}

	if out[0].ResolvedName != alias {
		t.Fatalf("resolved name несовпадение: получено=%q ожидалось=%q", out[0].ResolvedName, alias)
	}
}

func TestResolveExecutableToolCallsRejectsUndeclared(t *testing.T) {
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{Name: "web_search"},
		},
	}

	_, err := ResolveExecutableToolCalls(gp, []CohereActionRow{
		{
			ToolName:   "not_declared",
			Parameters: []byte(`{}`),
		},
	})

	if err == nil || !strings.Contains(err.Error(), "не объявлен") {
		t.Fatalf("ожидалось ошибка undeclared, получено: %v", err)
	}
}

func TestRunFnWithContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err := RunFnWithContext(ctx, func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 1, nil
	})

	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ожидался DeadlineExceeded, err=%v", err)
	}
}
