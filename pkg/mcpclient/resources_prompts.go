package mcpclient

import (
	"context"
	"encoding/json"
	"github.com/magomedcoder/gen/pkg/mcpclient/domain"
	"strings"

	"github.com/magomedcoder/gen/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DeclaredResource struct {
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
}

type DeclaredPrompt struct {
	Name          string
	Title         string
	Description   string
	ArgumentsJSON string
}

func ListResources(ctx context.Context, srv *domain.MCPServer) ([]DeclaredResource, error) {
	return listResources(ctx, srv, nil)
}

func listResources(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache) ([]DeclaredResource, error) {
	sid := int64(0)
	snm := ""
	if srv != nil {
		sid = srv.ID
		snm = strings.TrimSpace(srv.Name)
	}

	logger.D("MCP listResources: server_id=%d name=%q старт", sid, snm)

	var out []DeclaredResource
	err := withSession(ctx, srv, notify, func(cctx context.Context, session *mcp.ClientSession) error {
		var cursor string
		for {
			p := &mcp.ListResourcesParams{}
			if cursor != "" {
				p.Cursor = cursor
			}
			res, err := session.ListResources(cctx, p)
			if err != nil {
				return err
			}
			for _, r := range res.Resources {
				if r == nil || strings.TrimSpace(r.URI) == "" {
					continue
				}
				out = append(out, DeclaredResource{
					URI:         r.URI,
					Name:        r.Name,
					Title:       r.Title,
					Description: r.Description,
					MIMEType:    r.MIMEType,
				})
			}
			cursor = strings.TrimSpace(res.NextCursor)
			if cursor == "" {
				break
			}
		}
		return nil
	})
	if err != nil {
		logger.W("MCP listResources: server_id=%d name=%q err=%v", sid, snm, err)
	} else {
		logger.D("MCP listResources: server_id=%d name=%q всего=%d", sid, snm, len(out))
	}

	recordListResources(err)
	return out, err
}

func ListPrompts(ctx context.Context, srv *domain.MCPServer) ([]DeclaredPrompt, error) {
	return listPrompts(ctx, srv, nil)
}

func listPrompts(ctx context.Context, srv *domain.MCPServer, notify *ToolsListCache) ([]DeclaredPrompt, error) {
	sid := int64(0)
	snm := ""
	if srv != nil {
		sid = srv.ID
		snm = strings.TrimSpace(srv.Name)
	}

	logger.D("MCP listPrompts: server_id=%d name=%q старт", sid, snm)

	var out []DeclaredPrompt
	err := withSession(ctx, srv, notify, func(cctx context.Context, session *mcp.ClientSession) error {
		var cursor string
		for {
			p := &mcp.ListPromptsParams{}
			if cursor != "" {
				p.Cursor = cursor
			}
			res, err := session.ListPrompts(cctx, p)
			if err != nil {
				return err
			}
			for _, pr := range res.Prompts {
				if pr == nil || strings.TrimSpace(pr.Name) == "" {
					continue
				}
				argsJSON := "[]"
				if len(pr.Arguments) > 0 {
					b, err := json.Marshal(pr.Arguments)
					if err == nil {
						argsJSON = string(b)
					}
				}
				out = append(out, DeclaredPrompt{
					Name:          pr.Name,
					Title:         pr.Title,
					Description:   pr.Description,
					ArgumentsJSON: argsJSON,
				})
			}
			cursor = strings.TrimSpace(res.NextCursor)
			if cursor == "" {
				break
			}
		}
		return nil
	})

	if err != nil {
		logger.W("MCP listPrompts: server_id=%d name=%q err=%v", sid, snm, err)
	} else {
		logger.D("MCP listPrompts: server_id=%d name=%q всего=%d", sid, snm, len(out))
	}

	recordListPrompts(err)
	return out, err
}
