package modelresolve

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
)

func NormalizedAvailableModels(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, m := range raw {
		v := strings.TrimSpace(m)
		if v == "" {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}

func ResolveModelForUser(
	ctx context.Context,
	llmRepo domain.LLMRepository,
	requestedModel string,
	sessionModel string,
) (string, error) {
	availableRaw, err := llmRepo.GetModels(ctx)
	if err != nil {
		return "", err
	}

	available := NormalizedAvailableModels(availableRaw)
	if len(available) == 0 {
		return "", fmt.Errorf("нет доступных моделей на активных раннерах")
	}

	requested := strings.TrimSpace(requestedModel)
	if requested != "" {
		if slices.Contains(available, requested) {
			return requested, nil
		}

		return "", fmt.Errorf("модель %q недоступна (доступные: %s)", requested, strings.Join(available, ", "))
	}

	fromSession := strings.TrimSpace(sessionModel)
	if fromSession != "" && slices.Contains(available, fromSession) {
		return fromSession, nil
	}

	return available[0], nil
}
