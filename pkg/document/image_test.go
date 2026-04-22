package document

import (
	"encoding/base64"
	"testing"
)

func TestIsImageAttachment(t *testing.T) {
	if !IsImageAttachment("x.PNG") {
		t.Fatal("расширение PNG должно распознаваться как изображение")
	}

	if IsImageAttachment("doc.pdf") {
		t.Fatal("pdf не должен считаться изображением")
	}
}

func TestSniffStrictImageMIME_png(t *testing.T) {
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAKmKmTwAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatal(err)
	}

	got, err := SniffStrictImageMIME(raw)
	if err != nil {
		t.Fatal(err)
	}

	if got != "image/png" {
		t.Fatalf("got %q", got)
	}
}

func TestSniffStrictImageMIME_rejectsNonImage(t *testing.T) {
	if _, err := SniffStrictImageMIME([]byte("%PDF-1.4 minimal")); err == nil {
		t.Fatal("ожидалась ошибка для не-изображения")
	}
}

func TestImageMIMEFromFilename(t *testing.T) {
	if ImageMIMEFromFilename("a.JPEG") != "image/jpeg" {
		t.Fatal("jpeg")
	}

	if ImageMIMEFromFilename("x.pdf") != "" {
		t.Fatal("pdf")
	}
}
