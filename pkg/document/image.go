package document

import (
	"fmt"
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

func IsImageAttachment(filename string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	_, ok := imageExtensions[ext]
	return ok
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

	return nil
}
