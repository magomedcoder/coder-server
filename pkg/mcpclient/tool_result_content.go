package mcpclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxInlinedBase64Runes = 512 * 1024

func CallToolResultString(r *mcp.CallToolResult) string {
	if r == nil {
		return ""
	}

	var parts []string
	for _, c := range r.Content {
		if c == nil {
			continue
		}
		s := contentToLLMString(c)
		if strings.TrimSpace(s) != "" {
			parts = append(parts, s)
		}
	}

	if r.StructuredContent != nil {
		b, _ := json.Marshal(r.StructuredContent)
		if len(b) > 0 {
			parts = append(parts, string(b))
		}
	}

	if len(parts) == 0 && r.IsError {
		return "error инструмента (empty response)"
	}

	return strings.Join(parts, "\n")
}

func contentToLLMString(c mcp.Content) string {
	if c == nil {
		return ""
	}

	switch x := c.(type) {
	case *mcp.TextContent:
		return strings.TrimSpace(x.Text)
	case *mcp.ImageContent:
		return mcpImageContentToLLMPlaceholder(x.MIMEType, len(x.Data))
	case *mcp.AudioContent:
		return imageAudioToLLMString("Аудио", x.MIMEType, x.Data)
	case *mcp.ResourceLink:
		return resourceLinkToString(x)
	case *mcp.EmbeddedResource:
		if x.Resource == nil {
			return ""
		}
		return embeddedResourceToString(x.Resource)
	default:
		b, err := json.Marshal(c)
		if err == nil && len(b) > 0 {
			return string(b)
		}
		return fmt.Sprintf("%T", c)
	}
}

func mcpImageContentToLLMPlaceholder(mime string, n int) string {
	mime = strings.TrimSpace(mime)
	if n <= 0 {
		return fmt.Sprintf("[Изображение из инструмента mime=%q: пустые данные]", mime)
	}

	return fmt.Sprintf("[Изображение из инструмента mime=%q size_bytes=%d - пиксели не переданы в текст контекста; опирайтесь на вложение пользователя в чате при анализе кадра]", mime, n)
}

func imageAudioToLLMString(kind, mime string, data []byte) string {
	mime = strings.TrimSpace(mime)
	n := len(data)
	if n == 0 {
		return fmt.Sprintf("[%s mime=%q: пустые данные]", kind, mime)
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	if len(b64) > maxInlinedBase64Runes {
		return fmt.Sprintf(`[%s mime=%q base64_len=%d - данные обрезаны в выдаче; полный base64 слишком велик для контекста]`, kind, mime, len(b64))
	}

	return fmt.Sprintf("[%s mime=%q base64]\n%s", kind, mime, b64)
}

func resourceLinkToString(l *mcp.ResourceLink) string {
	if l == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[resource_link uri=%q", l.URI)
	if l.Name != "" {
		fmt.Fprintf(&b, " name=%q", l.Name)
	}
	if l.Title != "" {
		fmt.Fprintf(&b, " title=%q", l.Title)
	}
	if l.Description != "" {
		fmt.Fprintf(&b, " description=%q", l.Description)
	}
	if l.MIMEType != "" {
		fmt.Fprintf(&b, " mime=%q", l.MIMEType)
	}
	b.WriteString("]")
	return b.String()
}

func embeddedResourceToString(rc *mcp.ResourceContents) string {
	if rc == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[embedded resource uri=%q", rc.URI)
	if rc.MIMEType != "" {
		fmt.Fprintf(&b, " mime=%q", rc.MIMEType)
	}
	if rc.Text != "" {
		fmt.Fprintf(&b, "]\n%s", rc.Text)
		return b.String()
	}
	if len(rc.Blob) > 0 {
		mt := strings.ToLower(strings.TrimSpace(rc.MIMEType))
		if strings.HasPrefix(mt, "image/") {
			fmt.Fprintf(&b, " image_bytes=%d - пиксели не включены в контекст LLM]", len(rc.Blob))
			return b.String()
		}

		b64 := base64.StdEncoding.EncodeToString(rc.Blob)
		if len(b64) > maxInlinedBase64Runes {
			fmt.Fprintf(&b, " blob_bytes=%d - base64 обрезан в выдаче]", len(rc.Blob))
			return b.String()
		}
		fmt.Fprintf(&b, " base64]\n%s", b64)
		return b.String()
	}
	b.WriteString("]")
	return b.String()
}
