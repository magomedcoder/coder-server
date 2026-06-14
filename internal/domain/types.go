package domain

import (
	"github.com/magomedcoder/coder-server/pkg/context"
	"github.com/magomedcoder/coder-server/pkg/mcpregistry"
)

type WorkspaceContext = context.WorkspaceContext
type SelectionContext = context.SelectionContext
type ContextSnippet = context.ContextSnippet
type TreeEntry = context.TreeEntry
type EditorContext = context.EditorContext
type ChatContext = context.ChatContext

type MCPToolInfo = mcpregistry.ToolInfo
type MCPCallRequest = mcpregistry.CallRequest
