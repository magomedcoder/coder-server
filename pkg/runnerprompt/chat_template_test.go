package runnerprompt

import (
	"strings"
	"testing"

	runnertemplate "github.com/magomedcoder/gen-runner/template"
	"github.com/magomedcoder/gen/pkg/domain"
)

const sampleChatMLJinja = `{% for message in messages %}{{'<|im_start|>' + message['role'] + '\n' + message['content'] + '<|im_end|>' + '\n'}}{% endfor %}{% if add_generation_prompt %}{{ '<|im_start|>assistant\n' }}{% endif %}`

func TestBuildChatPrompt_chatml(t *testing.T) {
	msgs := []*domain.Message{
		domain.NewMessage(1, "Привет", domain.MessageRoleUser),
	}

	p, err := BuildChatPrompt(sampleChatMLJinja, msgs)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(p, "<|im_start|>user") || !strings.Contains(p, "Привет") {
		t.Fatalf("неожиданный промпт: %q", p)
	}

	if !strings.Contains(p, "<|im_start|>assistant") {
		t.Fatalf("нет заголовка assistant: %q", p)
	}
}

func TestRenderMatchedPreset_sameAsBuildChatPrompt(t *testing.T) {
	msgs := []*domain.Message{
		domain.NewMessage(1, "x", domain.MessageRoleUser),
	}

	a, err := BuildChatPrompt(sampleChatMLJinja, msgs)
	if err != nil {
		t.Fatal(err)
	}

	preset, err := runnertemplate.Named(strings.TrimSpace(sampleChatMLJinja))
	if err != nil {
		t.Fatal(err)
	}

	b, err := RenderMatchedPreset(preset, msgs)
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Fatalf("расхождение\na=%q\nb=%q", a, b)
	}
}

func TestPrepareRenderedPrompt_setsRendered(t *testing.T) {
	gp := WithClientChatTemplate(&domain.GenerationParams{}, sampleChatMLJinja)
	msgs := []*domain.Message{
		domain.NewMessage(1, "hi", domain.MessageRoleUser),
	}

	out, err := EnsureRenderedPrompt(gp, msgs)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(out.RenderedPrompt) == "" {
		t.Fatal("RenderedPrompt пуст")
	}

	if out.ChatTemplateJinja != sampleChatMLJinja {
		t.Fatalf("ChatTemplateJinja=%q", out.ChatTemplateJinja)
	}
}

func TestBuildChatPromptFromManifest_yaml(t *testing.T) {
	raw := []byte("template: |\n  " + strings.ReplaceAll(sampleChatMLJinja, "\n", "\n  "))
	m, err := ParseManifestYAML(raw)
	if err != nil {
		t.Fatal(err)
	}

	p, err := BuildChatPromptFromManifest(m, []*domain.Message{
		domain.NewMessage(1, "y", domain.MessageRoleUser),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(p, "y") {
		t.Fatalf("получено %q", p)
	}
}
