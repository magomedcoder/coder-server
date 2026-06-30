package document

import (
	"context"
	"strings"
)

func normalizeExtensionToken(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}

	if !strings.HasPrefix(s, ".") {
		return "." + s
	}

	return s
}

func IsPlainExtraExtension(filename string, extra []string) bool {
	want := normalizeExt(filename)
	if want == "" {
		return false
	}

	for _, e := range extra {
		if normalizeExtensionToken(e) == want {
			return true
		}
	}

	return false
}

func IsSupportedOrPlainExtra(filename string, extra []string) bool {
	if IsSupportedExtension(filename) {
		return true
	}

	return IsPlainExtraExtension(filename, extra)
}

func ExtractTextForRAGOrPlainExtra(ctx context.Context, filename string, content []byte, extraPlain []string) (text string, pdfPageRuneBounds []int, err error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	if IsSupportedExtension(filename) {
		return ExtractTextForRAGContext(ctx, filename, content)
	}

	if IsPlainExtraExtension(filename, extraPlain) {
		t, e := DecodeTextFileToUTF8(content)
		return t, nil, e
	}

	return "", nil, ErrUnsupportedAttachmentType
}
