package mcpclient

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCallToolResultString_TextAndImage(t *testing.T) {
	raw := []byte{0x89, 0x50}
	res := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "hello",
			},
			&mcp.ImageContent{
				MIMEType: "image/png",
				Data:     raw,
			},
		},
	}

	s := CallToolResultString(res)
	if !strings.Contains(s, "hello") {
		t.Fatalf("expected text, got %q", s)
	}

	if !strings.Contains(s, "image/png") || !strings.Contains(s, "size_bytes") {
		t.Fatalf("expected image placeholder, got %q", s)
	}

	if strings.Contains(s, "iVBOR") || strings.Contains(s, "base64]\n") {
		t.Fatalf("не ожидали встраивание base64 изображения в контекст: %q", s)
	}
}

func TestEmbeddedResourceImageBlob_placeholderNoBase64(t *testing.T) {
	s := embeddedResourceToString(&mcp.ResourceContents{
		URI:      "mem:test",
		MIMEType: "image/png",
		Blob:     []byte{1, 2, 3, 4, 5},
	})
	if strings.Contains(s, "base64") {
		t.Fatalf("неожиданный base64: %q", s)
	}

	if !strings.Contains(s, "image_bytes") {
		t.Fatalf("expected placeholder: %q", s)
	}
}

func TestResourceLinkToString(t *testing.T) {
	s := resourceLinkToString(&mcp.ResourceLink{
		URI:         "file:///x",
		Name:        "n",
		Description: "d",
		MIMEType:    "text/plain",
	})
	if !strings.Contains(s, "file:///x") {
		t.Fatal(s)
	}
}
