package mcpclient

import (
	"context"
	"errors"
	"github.com/magomedcoder/gen/pkg/mcpclient/domain"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ServerProbe struct {
	ProtocolVersion string
	ServerName      string
	ServerVersion   string
	Instructions    string
	HasTools        bool
	HasResources    bool
	HasPrompts      bool
}

func buildServerProbe(session *mcp.ClientSession) (*ServerProbe, error) {
	if session == nil {
		return nil, errors.New("nil session")
	}
	ir := session.InitializeResult()
	if ir == nil {
		return nil, errors.New("пустой InitializeResult")
	}

	p := &ServerProbe{
		ProtocolVersion: ir.ProtocolVersion,
		Instructions:    ir.Instructions,
	}
	if ir.ServerInfo != nil {
		p.ServerName = ir.ServerInfo.Name
		p.ServerVersion = ir.ServerInfo.Version
	}
	if caps := ir.Capabilities; caps != nil {
		p.HasTools = caps.Tools != nil
		p.HasResources = caps.Resources != nil
		p.HasPrompts = caps.Prompts != nil
	}
	return p, nil
}

func ProbeServer(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache) (*ServerProbe, error) {
	var out *ServerProbe
	err := withSession(ctx, srv, notify, func(_ context.Context, session *mcp.ClientSession) error {
		p, err := buildServerProbe(session)
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	recordProbe(err)
	if err != nil {
		return nil, err
	}
	return out, nil
}
