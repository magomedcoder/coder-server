package document

import "testing"

func TestIsImageAttachment(t *testing.T) {
	if !IsImageAttachment("x.PNG") {
		t.Fatal("расширение PNG должно распознаваться как изображение")
	}

	if IsImageAttachment("doc.pdf") {
		t.Fatal("pdf не должен считаться изображением")
	}
}
