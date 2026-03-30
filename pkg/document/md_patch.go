package document

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var (
	ErrMdPatchInvalidSpec     = errors.New("некорректная спецификация патча Markdown")
	ErrMdPatchAmbiguousSubstr = errors.New("уточнение replace_substring: вхождение должно быть ровно одно")
)

const (
	maxMdPatchSpecJSONBytes = 256 * 1024
	maxMdPatchOps           = 200
	maxMdPatchOutputRunes   = 600_000
	maxMdPatchLineCount     = 100_000
)

type mdPatchSpec struct {
	Ops []mdPatchOp `json:"ops"`
}

type mdPatchOp struct {
	Op    string   `json:"op"`
	Line  int      `json:"line,omitempty"`
	Count int      `json:"count,omitempty"`
	Lines []string `json:"lines,omitempty"`
	Text  string   `json:"text,omitempty"`
	Old   string   `json:"old,omitempty"`
	New   string   `json:"new,omitempty"`
}

func ApplyMarkdownPatchJSON(baseText string, patchJSON string) (string, error) {
	patchJSON = strings.TrimSpace(patchJSON)
	if patchJSON == "" {
		return "", fmt.Errorf("%w: пустой patch_json", ErrMdPatchInvalidSpec)
	}

	if len(patchJSON) > maxMdPatchSpecJSONBytes {
		return "", fmt.Errorf("%w: patch_json длиннее %d байт", ErrMdPatchInvalidSpec, maxMdPatchSpecJSONBytes)
	}

	if !utf8.ValidString(baseText) || !utf8.ValidString(patchJSON) {
		return "", fmt.Errorf("%w: ожидается UTF-8", ErrMdPatchInvalidSpec)
	}

	var spec mdPatchSpec
	if err := json.Unmarshal([]byte(patchJSON), &spec); err != nil {
		return "", fmt.Errorf("%w: %v", ErrMdPatchInvalidSpec, err)
	}

	if len(spec.Ops) == 0 {
		return "", fmt.Errorf("%w: нет операций ops", ErrMdPatchInvalidSpec)
	}

	if len(spec.Ops) > maxMdPatchOps {
		return "", fmt.Errorf("%w: слишком много операций (%d)", ErrMdPatchInvalidSpec, len(spec.Ops))
	}

	text := baseText
	for i, op := range spec.Ops {
		if err := validateMdPatchOutputSize(text); err != nil {
			return "", fmt.Errorf("операция #%d: %w", i+1, err)
		}

		var err error
		text, err = applyOneMdPatch(text, op)
		if err != nil {
			return "", fmt.Errorf("операция #%d (%s): %w", i+1, strings.TrimSpace(strings.ToLower(op.Op)), err)
		}
	}

	if err := validateMdPatchOutputSize(text); err != nil {
		return "", err
	}

	return text, nil
}

func validateMdPatchOutputSize(s string) error {
	if utf8.RuneCountInString(s) > maxMdPatchOutputRunes {
		return fmt.Errorf("%w: результат длиннее %d символов", ErrMdPatchInvalidSpec, maxMdPatchOutputRunes)
	}

	return nil
}

func applyOneMdPatch(text string, op mdPatchOp) (string, error) {
	kind := strings.TrimSpace(strings.ToLower(op.Op))
	switch kind {
	case "append":
		return text + op.Text, nil
	case "prepend":
		return op.Text + text, nil
	case "replace_substring":
		if op.Old == "" {
			return "", fmt.Errorf("%w: пустое поле old", ErrMdPatchInvalidSpec)
		}

		n := strings.Count(text, op.Old)
		if n == 0 {
			return "", fmt.Errorf("%w: подстрока не найдена", ErrMdPatchInvalidSpec)
		}

		if n > 1 {
			return "", ErrMdPatchAmbiguousSubstr
		}

		return strings.Replace(text, op.Old, op.New, 1), nil
	case "insert_before_line":
		lines := splitLines(text)
		if op.Line < 0 || op.Line > len(lines) {
			return "", fmt.Errorf("%w: line вне диапазона [0,%d]", ErrMdPatchInvalidSpec, len(lines))
		}

		insert := splitLines(op.Text)
		if len(lines)+len(insert) > maxMdPatchLineCount {
			return "", fmt.Errorf("%w: слишком много строк в документе", ErrMdPatchInvalidSpec)
		}

		out := make([]string, 0, len(lines)+len(insert))
		out = append(out, lines[:op.Line]...)
		out = append(out, insert...)
		out = append(out, lines[op.Line:]...)
		return joinLines(out), nil
	case "delete_line_range":
		if op.Count < 1 {
			return "", fmt.Errorf("%w: count должен быть >= 1", ErrMdPatchInvalidSpec)
		}

		lines := splitLines(text)
		if op.Line < 0 || op.Line+op.Count > len(lines) {
			return "", fmt.Errorf("%w: delete выходит за пределы строк", ErrMdPatchInvalidSpec)
		}

		out := append(lines[:op.Line], lines[op.Line+op.Count:]...)
		return joinLines(out), nil
	case "replace_line_range":
		if op.Count < 1 {
			return "", fmt.Errorf("%w: count должен быть >= 1", ErrMdPatchInvalidSpec)
		}

		lines := splitLines(text)
		if op.Line < 0 || op.Line+op.Count > len(lines) {
			return "", fmt.Errorf("%w: replace выходит за пределы строк", ErrMdPatchInvalidSpec)
		}

		if len(lines)-op.Count+len(op.Lines) > maxMdPatchLineCount {
			return "", fmt.Errorf("%w: слишком много строк", ErrMdPatchInvalidSpec)
		}

		out := make([]string, 0, len(lines)-op.Count+len(op.Lines))
		out = append(out, lines[:op.Line]...)
		out = append(out, op.Lines...)
		out = append(out, lines[op.Line+op.Count:]...)
		return joinLines(out), nil
	default:
		if kind == "" {
			return "", fmt.Errorf("%w: пустое поле op", ErrMdPatchInvalidSpec)
		}
		return "", fmt.Errorf("%w: неизвестная операция %q", ErrMdPatchInvalidSpec, op.Op)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}

	return strings.Split(s, "\n")
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
