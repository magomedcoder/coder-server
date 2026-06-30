package document

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

var imageExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
	".gif":  {},
}

var allowedChatImageMIMETypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

func IsImageAttachment(filename string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	_, ok := imageExtensions[ext]
	return ok
}

func IsAllowedChatImageMIME(mime string) bool {
	mime = strings.ToLower(strings.TrimSpace(mime))
	_, ok := allowedChatImageMIMETypes[mime]
	return ok
}

func ImageMIMEFromFilename(filename string) string {
	if !IsImageAttachment(filename) {
		return ""
	}

	switch strings.ToLower(filepath.Ext(strings.TrimSpace(filename))) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func SniffStrictImageMIME(content []byte) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("пустое вложение")
	}

	ct := http.DetectContentType(content)
	if !IsAllowedChatImageMIME(ct) {
		return "", fmt.Errorf("содержимое не похоже на допустимое изображение (получено %q)", ct)
	}

	return strings.ToLower(strings.TrimSpace(ct)), nil
}

func ValidateImageAttachment(filename string, content []byte) error {
	if len(content) == 0 {
		return fmt.Errorf("пустое вложение")
	}

	if len(content) > MaxRecommendedAttachmentSizeBytes {
		return fmt.Errorf("размер изображения превышает лимит %d байт", MaxRecommendedAttachmentSizeBytes)
	}

	if !IsImageAttachment(filename) {
		return ErrUnsupportedAttachmentType
	}

	if _, err := SniffStrictImageMIME(content); err != nil {
		return err
	}

	return nil
}
