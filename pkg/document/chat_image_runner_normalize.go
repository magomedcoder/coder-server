package document

import (
	"bytes"
	"fmt"
	"image"
	stdraw "image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const MaxRunnerChatImageEdge = 2048
const runnerNormalizeJPEGQuality = 85

func NormalizeChatImageBytesForRunner(src []byte, sniffedMIME string) ([]byte, string, error) {
	if len(src) == 0 {
		return nil, "", fmt.Errorf("пустое изображение")
	}

	mime := strings.ToLower(strings.TrimSpace(sniffedMIME))
	if mime != "" && !IsAllowedChatImageMIME(mime) {
		return nil, "", fmt.Errorf("недопустимый MIME для нормализации: %q", sniffedMIME)
	}

	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, "", fmt.Errorf("декод изображения: %w", err)
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, "", fmt.Errorf("некорректные размеры изображения")
	}

	if maxInt(w, h) <= MaxRunnerChatImageEdge {
		if mime == "" {
			m2, e := SniffStrictImageMIME(src)
			if e != nil {
				return nil, "", e
			}

			mime = m2
		}

		return src, mime, nil
	}

	nw, nh := scaleToFitMaxEdge(w, h, MaxRunnerChatImageEdge)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, stdraw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: runnerNormalizeJPEGQuality}); err != nil {
		return nil, "", fmt.Errorf("jpeg encode: %w", err)
	}

	return buf.Bytes(), "image/jpeg", nil
}

func FilenameForNormalizedImageMime(origName, newMIME string) string {
	origName = strings.TrimSpace(origName)
	newMIME = strings.ToLower(strings.TrimSpace(newMIME))
	if origName == "" {
		switch newMIME {
		case "image/jpeg":
			return "image.jpg"
		case "image/png":
			return "image.png"
		case "image/webp":
			return "image.webp"
		case "image/gif":
			return "image.gif"
		default:
			return "image.bin"
		}
	}

	oldExt := strings.ToLower(filepath.Ext(origName))
	base := strings.TrimSuffix(origName, filepath.Ext(origName))

	switch newMIME {
	case "image/jpeg":
		if oldExt == ".jpg" || oldExt == ".jpeg" {
			return origName
		}

		return base + ".jpg"
	case "image/png":
		if oldExt == ".png" {
			return origName
		}

		return base + ".png"
	case "image/webp":
		if oldExt == ".webp" {
			return origName
		}

		return base + ".webp"
	case "image/gif":
		if oldExt == ".gif" {
			return origName
		}

		return base + ".gif"
	default:
		return origName
	}
}

func scaleToFitMaxEdge(w, h, maxEdge int) (int, int) {
	m := maxInt(w, h)
	if m <= maxEdge {
		return w, h
	}

	r := float64(maxEdge) / float64(m)
	nw := int(math.Round(float64(w) * r))
	nh := int(math.Round(float64(h) * r))
	if nw < 1 {
		nw = 1
	}

	if nh < 1 {
		nh = 1
	}

	return nw, nh
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
