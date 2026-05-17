package toolloop

import (
	"encoding/json"
	"fmt"
	"strings"
)

func MustStringField(m map[string]json.RawMessage, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("отсутствует поле %q", key)
	}

	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", fmt.Errorf("поле %q: ожидается строка", key)
	}

	return strings.TrimSpace(s), nil
}

func OptionalStringField(m map[string]json.RawMessage, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}

	var s string
	_ = json.Unmarshal(v, &s)
	return strings.TrimSpace(s)
}

func OptionalInt64Field(m map[string]json.RawMessage, key string) (int64, bool, error) {
	v, ok := m[key]
	if !ok {
		return 0, false, nil
	}

	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, false, err
	}

	return int64(f), true, nil
}

func OptionalInt32Field(m map[string]json.RawMessage, key string) (int32, bool, error) {
	v, ok := m[key]
	if !ok {
		return 0, false, nil
	}

	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, false, err
	}

	return int32(f), true, nil
}

func FilterExecutableToolRows(rows []CohereActionRow) []CohereActionRow {
	var out []CohereActionRow
	for _, r := range rows {
		if !IsDirectAnswerTool(r.ToolName) {
			out = append(out, r)
		}
	}

	return out
}
