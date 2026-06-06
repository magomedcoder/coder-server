package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
	"gopkg.in/yaml.v3"
)

type ModelfileManifest struct {
	From     string `yaml:"from,omitempty"`
	System   string `yaml:"system,omitempty"`
	Template string `yaml:"template,omitempty"`
}

func ParseManifestYAML(data []byte) (*ModelfileManifest, error) {
	var m ModelfileManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

func LoadManifestYAML(path string) (*ModelfileManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseManifestYAML(data)
}

func ChatTemplateFromManifest(m *ModelfileManifest) string {
	if m == nil {
		return ""
	}

	return strings.TrimSpace(m.Template)
}

func BuildChatPromptFromManifest(m *ModelfileManifest, messages []*domain.Message) (string, error) {
	tmpl := ChatTemplateFromManifest(m)
	if tmpl == "" {
		return "", fmt.Errorf("runnerprompt: в манифесте нет template")
	}

	return BuildChatPrompt(tmpl, messages)
}

func WithClientChatTemplate(gp *domain.GenerationParams, chatTemplateJinja string) *domain.GenerationParams {
	if gp == nil {
		return &domain.GenerationParams{ChatTemplateJinja: strings.TrimSpace(chatTemplateJinja)}
	}

	out := *gp
	out.ChatTemplateJinja = strings.TrimSpace(chatTemplateJinja)

	return &out
}

func PrepareRenderedPrompt(gp *domain.GenerationParams, messages []*domain.Message) (*domain.GenerationParams, error) {
	if gp == nil {
		return nil, nil
	}

	if rp := strings.TrimSpace(gp.RenderedPrompt); rp != "" {
		return gp, nil
	}

	tmpl := strings.TrimSpace(gp.ChatTemplateJinja)
	if tmpl == "" {
		return gp, nil
	}

	prompt, err := BuildChatPrompt(tmpl, messages)
	if err != nil {
		return nil, err
	}

	return WithRenderedPrompt(gp, prompt), nil
}

func EnsureRenderedPrompt(gp *domain.GenerationParams, messages []*domain.Message) (*domain.GenerationParams, error) {
	if gp == nil {
		return nil, nil
	}

	return PrepareRenderedPrompt(gp, PrepareMessagesForRunner(messages))
}
