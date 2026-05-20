package runnerprompt

import (
	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/mcpprompt"
)

type SystemToolsOptions struct {
	Tools      []domain.Tool
	MCPEntries []mcpprompt.ServerEntry
}

func EnrichSystemMessage(msg *domain.Message, opts SystemToolsOptions) {
	if msg == nil {
		return
	}

	if block := mcpprompt.BuildSessionHints(opts.MCPEntries); block != "" {
		msg.Content = mcpprompt.AppendBlock(msg.Content, block)
	}

	if block := mcpprompt.BuildToolCatalog(opts.Tools); block != "" {
		msg.Content = mcpprompt.AppendBlock(msg.Content, block)
	}

	AppendToolsInvocationToSystem(msg, opts.Tools)
}
