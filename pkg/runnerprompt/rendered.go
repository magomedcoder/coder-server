package runnerprompt

import "github.com/magomedcoder/gen/pkg/domain"

func WithRenderedPrompt(gp *domain.GenerationParams, prompt string) *domain.GenerationParams {
	if gp == nil {
		return &domain.GenerationParams{
			RenderedPrompt: prompt,
		}
	}

	out := *gp
	out.RenderedPrompt = prompt

	return &out
}
