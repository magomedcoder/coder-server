package toolloop

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/magomedcoder/gen/pkg/chatstream"
	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/llmrunner"
	"github.com/magomedcoder/gen/pkg/logger"
	"github.com/magomedcoder/gen/pkg/mcpclient"
	"github.com/magomedcoder/gen/pkg/prompt"
)

func SamplingGenParamsForMCP(gp *domain.GenerationParams) *domain.GenerationParams {
	if gp == nil {
		return &domain.GenerationParams{}
	}

	out := *gp
	out.Tools = nil
	out.ResponseFormat = nil
	out.RenderedPrompt = ""
	out.ChatTemplateJinja = ""

	return &out
}

func CloneGenParamsForToolCalls(in *domain.GenerationParams) *domain.GenerationParams {
	if in == nil {
		return nil
	}

	out := *in
	out.ResponseFormat = nil
	out.RenderedPrompt = ""
	out.ChatTemplateJinja = ""

	return &out
}

func RunnerInferenceParams(in *domain.GenerationParams, messages []*domain.Message) *domain.GenerationParams {
	if in == nil {
		return nil
	}

	out := *in
	out.Tools = nil
	out.ResponseFormat = nil
	out.RenderedPrompt = ""
	out.ChatTemplateJinja = ""

	if rp := strings.TrimSpace(in.RenderedPrompt); rp != "" {
		out.RenderedPrompt = rp
		return &out
	}

	tmpl := strings.TrimSpace(in.ChatTemplateJinja)
	if tmpl == "" {
		return &out
	}

	prepared := prompt.PrepareMessagesForRunner(messages)
	if p, err := prompt.BuildChatPrompt(tmpl, prepared); err == nil {
		out.RenderedPrompt = p
	}

	return &out
}

func AllowedToolNameSet(tools []domain.Tool) map[string]struct{} {
	m := make(map[string]struct{})
	for _, t := range tools {
		n := NormalizeToolName(t.Name)
		if n != "" {
			m[n] = struct{}{}
		}
	}

	return m
}

var mcpAliasFromModelRe = regexp.MustCompile(`^mcp_(\d+)_h([0-9a-f]+)$`)

func TryRecoverSingleMCPServerToolAlias(genParams *domain.GenerationParams, n string) (canon string, serverID int64, ok bool) {
	if genParams == nil {
		return "", 0, false
	}

	m := mcpAliasFromModelRe.FindStringSubmatch(n)
	if len(m) != 3 {
		return "", 0, false
	}

	sid, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || sid <= 0 {
		return "", 0, false
	}

	var canonList []string
	for _, t := range genParams.Tools {
		c := NormalizeToolName(t.Name)
		if c == "" {
			continue
		}

		tsid, _, ok := mcpclient.ParseToolAlias(c)
		if !ok || tsid != sid {
			continue
		}

		canonList = append(canonList, c)
	}

	if len(canonList) != 1 {
		return "", sid, false
	}

	return canonList[0], sid, true
}

func LogToolResolveMismatch(genParams *domain.GenerationParams, requested string) {
	n := NormalizeToolName(requested)
	var mcpNames []string
	if genParams != nil {
		for _, t := range genParams.Tools {
			c := NormalizeToolName(t.Name)
			if strings.HasPrefix(c, "mcp_") {
				mcpNames = append(mcpNames, c)
			}
		}
	}

	const maxList = 24
	list := strings.Join(mcpNames, ", ")
	if len(mcpNames) > maxList {
		list = strings.Join(mcpNames[:maxList], ", ") + fmt.Sprintf(" ...(всего mcp_*=%d)", len(mcpNames))
	}

	logger.W("ChatUseCase tool-loop: phase=resolve_tools_mismatch requested=%q normalized=%q mcp_declared_count=%d declared_mcp_sample=[%s]", requested, n, len(mcpNames), list)
	if sid, orig, ok := mcpclient.ParseToolAlias(n); ok {
		logger.W("ChatUseCase tool-loop: phase=resolve_tools_mismatch decoded server_id=%d name_from_model_hex=%q", sid, orig)
	} else if mcpAliasFromModelRe.MatchString(n) {
		logger.W("ChatUseCase tool-loop: phase=resolve_tools_mismatch looks_like_mcp_alias_but_hex_decode_failed normalized=%q", n)
	}
}

func ResolveDeclaredToolName(genParams *domain.GenerationParams, raw string) (string, bool) {
	n := NormalizeToolName(raw)
	if genParams == nil || n == "" {
		return "", false
	}

	allowed := AllowedToolNameSet(genParams.Tools)
	if _, ok := allowed[n]; ok {
		return n, true
	}

	type cand struct {
		sid   int64
		canon string
	}

	var hits []cand
	for _, t := range genParams.Tools {
		canon := NormalizeToolName(t.Name)
		if canon == "" {
			continue
		}
		if _, ok := allowed[canon]; !ok {
			continue
		}

		sid, orig, ok := mcpclient.ParseToolAlias(canon)
		if !ok || sid <= 0 {
			continue
		}

		if NormalizeToolName(orig) == n {
			hits = append(hits, cand{sid: sid, canon: canon})
		}
	}

	if len(hits) == 0 {
		if canon, sid, ok := TryRecoverSingleMCPServerToolAlias(genParams, n); ok {
			logger.W("ChatUseCase tool-loop: phase=mcp_alias_recovered server_id=%d requested=%q resolved=%q (на сервере один MCP-tool; подменён неверный h... суффикс)", sid, strings.TrimSpace(raw), canon)
			return canon, true
		}
		return "", false
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].sid != hits[j].sid {
			return hits[i].sid < hits[j].sid
		}

		return hits[i].canon < hits[j].canon
	})

	return hits[0].canon, true
}

func NormalizeToolName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")

	return s
}

func DrainLLMStreamChannelForward(ch chan domain.LLMStreamChunk, forward func(c domain.LLMStreamChunk) bool) (raw string, streamedNonEmpty bool) {
	var b strings.Builder
	for c := range ch {
		if c.Content != "" {
			b.WriteString(c.Content)
		}
		if c.Content == "" && c.ReasoningContent == "" {
			continue
		}

		if !forward(c) {
			for c2 := range ch {
				b.WriteString(c2.Content)
			}

			return b.String(), true
		}

		streamedNonEmpty = true
	}

	return b.String(), streamedNonEmpty
}

func StreamToolRoundComplete(send func(chatstream.ChatStreamChunk) bool, messageID int64, streamed bool, modelFullTrimmed, canonical string) {
	if !streamed {
		_ = send(chatstream.ChatStreamChunk{
			Kind:      chatstream.StreamChunkKindText,
			Text:      canonical,
			MessageID: messageID,
		})
		return
	}

	if canonical == modelFullTrimmed {
		_ = send(chatstream.ChatStreamChunk{
			Kind:      chatstream.StreamChunkKindText,
			Text:      "",
			MessageID: messageID,
		})
		return
	}

	_ = send(chatstream.ChatStreamChunk{
		Kind:      chatstream.StreamChunkKindText,
		Text:      canonical,
		MessageID: messageID,
	})
}

var reActionJSON = regexp.MustCompile("(?is)(?:Action|Действие):\\s*" + "```" + `json\s*([\s\S]*?)` + "```")

func ExtractCohereActionJSON(text string) string {
	m := reActionJSON.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}

	return strings.TrimSpace(m[1])
}

func ExtractFirstFencedToolArray(text string) string {
	s := text
	for len(s) > 0 {
		start := strings.Index(s, "```")
		if start < 0 {
			return ""
		}

		afterOpen := s[start+3:]
		bodyStart := 0
		if nl := strings.IndexByte(afterOpen, '\n'); nl >= 0 {
			first := strings.TrimSpace(afterOpen[:nl])
			if len(first) > 0 && !strings.ContainsAny(first, " \t") {
				bodyStart = nl + 1
			}
		}

		rest := afterOpen[bodyStart:]
		before, _, ok := strings.Cut(rest, "```")
		if !ok {
			return ""
		}

		raw := strings.TrimSpace(before)
		tr := strings.TrimSpace(raw)
		if strings.HasPrefix(tr, "[") || strings.HasPrefix(tr, "{") {
			if rows, err := ParseCohereActionList(raw); err == nil && len(rows) > 0 && ToolActionRowsHaveNames(rows) {
				return raw
			}
		}

		s = afterOpen
	}

	return ""
}

func ExtractFirstJSONArray(text string) string {
	_, after, ok := strings.Cut(text, "```json")
	if !ok {
		return ""
	}

	rest := after
	before, _, ok := strings.Cut(rest, "```")
	if !ok {
		return ""
	}

	raw := strings.TrimSpace(before)
	tr := strings.TrimSpace(raw)
	if !strings.HasPrefix(tr, "[") && !strings.HasPrefix(tr, "{") {
		return ""
	}

	if rows, err := ParseCohereActionList(raw); err != nil || len(rows) == 0 || !ToolActionRowsHaveNames(rows) {
		return ""
	}

	return raw
}

func ExtractLeadingJSONArray(text string) string {
	s := strings.TrimSpace(text)
	if len(s) == 0 || s[0] != '[' {
		return ""
	}

	depth := 0
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if escape {
			escape = false
			continue
		}

		if inString {
			if b == '\\' {
				escape = true
			} else if b == '"' {
				inString = false
			}

			continue
		}

		switch b {
		case '"':
			inString = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}

	return ""
}

func ExtractLeadingJSONObject(text string) string {
	s := strings.TrimSpace(text)
	if len(s) == 0 || s[0] != '{' {
		return ""
	}

	depth := 0
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if escape {
			escape = false
			continue
		}

		if inString {
			if b == '\\' {
				escape = true
			} else if b == '"' {
				inString = false
			}

			continue
		}

		switch b {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}

	return ""
}

func ExtractEmbeddedJSONArray(text string) string {
	s := text
	for {
		idx := strings.Index(s, "[")
		if idx < 0 {
			return ""
		}

		sub := s[idx:]
		candidate := ExtractLeadingJSONArray(sub)
		if candidate != "" {
			rows, err := ParseCohereActionList(candidate)
			if err == nil && len(rows) > 0 && ToolActionRowsHaveNames(rows) {
				return candidate
			}
		}

		s = s[idx+1:]
	}
}

func ExtractEmbeddedJSONObject(text string) string {
	s := text
	for {
		idx := strings.Index(s, "{")
		if idx < 0 {
			return ""
		}

		sub := s[idx:]
		candidate := ExtractLeadingJSONObject(sub)
		if candidate != "" {
			rows, err := ParseCohereActionList(candidate)
			if err == nil && len(rows) > 0 && ToolActionRowsHaveNames(rows) {
				return candidate
			}
		}

		s = s[idx+1:]
	}
}

func ToolActionRowsHaveNames(rows []CohereActionRow) bool {
	for _, r := range rows {
		if strings.TrimSpace(r.ToolName) != "" {
			return true
		}
	}

	return false
}

func ExtractToolActionBlob(text string) string {
	if s := ExtractCohereActionJSON(text); s != "" {
		return s
	}

	if s := ExtractFirstJSONArray(text); s != "" {
		return s
	}

	if s := ExtractFirstFencedToolArray(text); s != "" {
		return s
	}

	if s := ExtractLeadingJSONArray(text); s != "" {
		return s
	}

	if s := ExtractLeadingJSONObject(text); s != "" {
		if rows, err := ParseCohereActionList(s); err == nil && len(rows) > 0 && ToolActionRowsHaveNames(rows) {
			return s
		}
	}

	if s := ExtractEmbeddedJSONArray(text); s != "" {
		return s
	}

	return ExtractEmbeddedJSONObject(text)
}

func ParseCohereActionList(blob string) ([]CohereActionRow, error) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return nil, nil
	}

	var asSlice []CohereActionRow
	if err := json.Unmarshal([]byte(blob), &asSlice); err == nil {
		if len(asSlice) > 0 {
			return NormalizeToolActionRows(asSlice)
		}

		if strings.HasPrefix(strings.TrimSpace(blob), "[") {
			return nil, fmt.Errorf("пустой список вызовов инструментов")
		}
	}

	type legacyNameArgs struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	var legacy legacyNameArgs
	if err := json.Unmarshal([]byte(blob), &legacy); err == nil && strings.TrimSpace(legacy.Name) != "" {
		args := legacy.Arguments
		if len(args) == 0 || string(args) == "null" {
			args = json.RawMessage(`{}`)
		}

		return NormalizeToolActionRows([]CohereActionRow{{
			ToolName:   strings.TrimSpace(legacy.Name),
			Parameters: args,
		}})
	}

	type legacyToolArgs struct {
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
	}
	var legacyTool legacyToolArgs
	if err := json.Unmarshal([]byte(blob), &legacyTool); err == nil && strings.TrimSpace(legacyTool.Tool) != "" {
		args := legacyTool.Arguments
		if len(args) == 0 || string(args) == "null" {
			args = json.RawMessage(`{}`)
		}

		return NormalizeToolActionRows([]CohereActionRow{{
			ToolName:   strings.TrimSpace(legacyTool.Tool),
			Parameters: args,
		}})
	}

	type legacyToolParams struct {
		ToolName   string          `json:"tool_name"`
		Parameters json.RawMessage `json:"parameters"`
	}
	var tp legacyToolParams
	if err := json.Unmarshal([]byte(blob), &tp); err == nil && strings.TrimSpace(tp.ToolName) != "" {
		args := tp.Parameters
		if len(args) == 0 || string(args) == "null" {
			args = json.RawMessage(`{}`)
		}

		return NormalizeToolActionRows([]CohereActionRow{{
			ToolName:   strings.TrimSpace(tp.ToolName),
			Parameters: args,
		}})
	}

	return nil, fmt.Errorf("неверный формат вызова инструментов (ожидается JSON-массив с tool_name/parameters или объект name/arguments/tool/arguments)")
}

func NormalizeToolActionRows(rows []CohereActionRow) ([]CohereActionRow, error) {
	out := make([]CohereActionRow, 0, len(rows))
	for _, row := range rows {
		params, err := NormalizeToolParameters(row.Parameters)
		if err != nil {
			return nil, fmt.Errorf("arguments для %q: %w", strings.TrimSpace(row.ToolName), err)
		}

		out = append(out, CohereActionRow{
			ToolName:   strings.TrimSpace(row.ToolName),
			Parameters: params,
		})
	}

	return out, nil
}

func NormalizeToolParameters(raw json.RawMessage) (json.RawMessage, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(`{}`), nil
	}

	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return nil, err
		}

		encoded = strings.TrimSpace(encoded)
		if encoded == "" {
			return json.RawMessage(`{}`), nil
		}

		raw = json.RawMessage(encoded)
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(b), nil
}

func IsDirectAnswerTool(name string) bool {
	switch NormalizeToolName(name) {
	case
		"directly_answer",
		"directlyanswer":
		return true
	default:
		return false
	}
}

func DirectAnswerText(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return strings.TrimSpace(string(params))
	}

	for _, key := range []string{"answer", "text", "message", "content"} {
		if v, ok := m[key]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				return strings.TrimSpace(s)
			}
		}
	}

	return strings.TrimSpace(string(params))
}

func ToolCallsToOpenAIJSON(calls []CohereActionRow) (string, error) {
	type fn struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}

	type item struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function fn     `json:"function"`
	}

	out := make([]item, 0, len(calls))
	for i, c := range calls {
		id := fmt.Sprintf("call_%d", i+1)
		args := strings.TrimSpace(string(c.Parameters))
		if args == "" {
			args = "{}"
		}

		out = append(out, item{
			ID:   id,
			Type: "function",
			Function: fn{
				Name:      c.ToolName,
				Arguments: args,
			},
		})
	}

	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func MaxToolInvocationRounds(reg *llmrunner.Registry) int {
	n := DefaultToolLoopRounds
	if reg != nil {
		if hinted := reg.AggregateChatHints().MaxToolInvocationRounds; hinted > 0 {
			n = hinted
		}
	}
	if n < 1 {
		return 1
	}
	if n > MaxToolLoopRoundsCap {
		return MaxToolLoopRoundsCap
	}
	return n
}

func ResolveExecutableToolCalls(genParams *domain.GenerationParams, rows []CohereActionRow) ([]ExecutableToolCall, error) {
	out := make([]ExecutableToolCall, 0, len(rows))
	for _, row := range rows {
		resolved, ok := ResolveDeclaredToolName(genParams, row.ToolName)
		if !ok {
			LogToolResolveMismatch(genParams, row.ToolName)
			return nil, fmt.Errorf("инструмент %q не объявлен в настройках сессии (имя должно совпасть с одним из tools; для MCP - точный алиас mcp_<id>_h<hex> или короткое имя инструмента на сервере)", row.ToolName)
		}
		out = append(out, ExecutableToolCall{
			RequestedName: row.ToolName,
			ResolvedName:  resolved,
			Parameters:    row.Parameters,
		})
	}
	return out, nil
}

func ExecutableCallsToActionRows(in []ExecutableToolCall) []CohereActionRow {
	out := make([]CohereActionRow, 0, len(in))
	for _, c := range in {
		out = append(out, CohereActionRow{
			ToolName:   c.ResolvedName,
			Parameters: c.Parameters,
		})
	}
	return out
}

func TruncateToolResult(s string) string {
	if utf8.RuneCountInString(s) <= MaxToolResultRunes {
		return s
	}
	r := []rune(s)

	return string(r[:MaxToolResultRunes]) + "\n...(обрезано)"
}
func WebSearchToolDefinition() domain.Tool {
	return domain.Tool{
		Name:           "web_search",
		Description:    "Поиск актуальной информации в интернете: новости, цены, погода, документация, свежие факты. Используй, когда response зависит от текущих данных или знания модели могут быть устаревшими. Передай короткий точный request на языке пользователя или на английском.",
		ParametersJSON: `{"type":"object","properties":{"query":{"type":"string","description":"Поисковый request: несколько ключевых слов или короткая фраза."}},"required":["query"]}`,
	}
}

func ToolCallNamesForLog(calls []ExecutableToolCall) string {
	if len(calls) == 0 {
		return ""
	}

	parts := make([]string, 0, len(calls))
	for i, c := range calls {
		if i >= 12 {
			parts = append(parts, fmt.Sprintf("...+%d", len(calls)-12))
			break
		}

		rn := strings.TrimSpace(c.RequestedName)
		if rn == "" {
			rn = "(?)"
		}

		parts = append(parts, rn)
	}

	return strings.Join(parts, ",")
}
