package document

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestNormalizeChatImageBytesForRunner_smallUnchanged(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	src := buf.Bytes()
	out, mime, err := NormalizeChatImageBytesForRunner(src, "image/png")
	if err != nil {
		t.Fatal(err)
	}

	if mime != "image/png" || !bytes.Equal(out, src) {
		t.Fatalf("mime=%q same=%t", mime, bytes.Equal(out, src))
	}
}

func TestNormalizeChatImageBytesForRunner_downscaleToJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2600, 20))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	out, mime, err := NormalizeChatImageBytesForRunner(buf.Bytes(), "image/png")
	if err != nil {
		t.Fatal(err)
	}

	if mime != "image/jpeg" {
		t.Fatalf("mime=%q", mime)
	}

	img2, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}

	b := img2.Bounds()
	if maxInt(b.Dx(), b.Dy()) > MaxRunnerChatImageEdge {
		t.Fatalf("bounds too large: %dx%d", b.Dx(), b.Dy())
	}
}

func TestFilenameForNormalizedImageMime_pngToJpeg(t *testing.T) {
	got := FilenameForNormalizedImageMime("photo.PNG", "image/jpeg")
	if got != "photo.jpg" {
		t.Fatalf("%q", got)
	}
}
