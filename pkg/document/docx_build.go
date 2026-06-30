package document

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

var (
	ErrDocxBuildInvalidSpec = errors.New("некорректная спецификация DOCX")
	ErrDocxBuildTooLarge    = errors.New("размер документа DOCX превышает лимит")
)

const (
	maxDocxBuildSpecJSONBytes = 256 * 1024
	maxDocxParagraphs         = 4000
	maxDocxRunesPerParagraph  = 16_000
)

type docxBuildSpec struct {
	Paragraphs []string `json:"paragraphs"`
	Title      string   `json:"title,omitempty"`
}

func BuildDOCXFromSpecJSON(specJSON string) ([]byte, error) {
	specJSON = strings.TrimSpace(specJSON)
	if specJSON == "" {
		return nil, fmt.Errorf("%w: пустой spec_json", ErrDocxBuildInvalidSpec)
	}

	if len(specJSON) > maxDocxBuildSpecJSONBytes {
		return nil, fmt.Errorf("%w: spec_json длиннее %d байт", ErrDocxBuildInvalidSpec, maxDocxBuildSpecJSONBytes)
	}

	var spec docxBuildSpec
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDocxBuildInvalidSpec, err)
	}

	var paras []string
	if t := strings.TrimSpace(spec.Title); t != "" {
		paras = append(paras, t)
	}

	paras = append(paras, spec.Paragraphs...)

	if len(paras) == 0 {
		return nil, fmt.Errorf("%w: нет ни title, ни paragraphs", ErrDocxBuildInvalidSpec)
	}

	if len(paras) > maxDocxParagraphs {
		return nil, fmt.Errorf("%w: слишком много абзацев (%d)", ErrDocxBuildInvalidSpec, len(paras))
	}

	for i, p := range paras {
		if !utf8.ValidString(p) {
			return nil, fmt.Errorf("%w: абзац %d не UTF-8", ErrDocxBuildInvalidSpec, i)
		}

		if utf8.RuneCountInString(p) > maxDocxRunesPerParagraph {
			return nil, fmt.Errorf("%w: абзац %d слишком длинный", ErrDocxBuildInvalidSpec, i)
		}
	}

	docXML, err := buildDocumentXML(paras)
	if err != nil {
		return nil, err
	}

	if len(docXML) > MaxRecommendedAttachmentSizeBytes {
		return nil, ErrDocxBuildTooLarge
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := writeZipEntry(zw, "[Content_Types].xml", contentTypesXML); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := writeZipEntry(zw, "_rels/.rels", rootRelsXML); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := writeZipEntry(zw, "word/_rels/document.xml.rels", documentRelsXML); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := writeZipEntry(zw, "word/document.xml", docXML); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("docx zip: %w", err)
	}

	out := buf.Bytes()
	if len(out) > MaxRecommendedAttachmentSizeBytes {
		return nil, ErrDocxBuildTooLarge
	}

	return out, nil
}

func writeZipEntry(zw *zip.Writer, name, body string) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("zip %s: %w", name, err)
	}

	if _, err := io.WriteString(w, body); err != nil {
		return fmt.Errorf("zip %s: %w", name, err)
	}

	return nil
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>
`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>
`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>
`

func buildDocumentXML(paragraphs []string) (string, error) {
	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, raw := range paragraphs {
		writeWordParagraph(&b, sanitizeXMLText(raw))
	}
	b.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String(), nil
}

func sanitizeXMLText(s string) string {
	var out strings.Builder
	for _, r := range s {
		switch {
		case r == 0x9, r == 0xA, r == 0xD:
			out.WriteRune(r)
		case r >= 0x20 && r <= 0xD7FF, r >= 0xE000 && r <= 0xFFFD, r >= 0x10000 && r <= 0x10FFFF:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func writeWordParagraph(b *strings.Builder, text string) {
	b.WriteString("<w:p>")
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		if i > 0 {
			b.WriteString("<w:r><w:br/></w:r>")
		}

		b.WriteString("<w:r><w:t")
		if preserveSpaceNeeded(part) {
			b.WriteString(` xml:space="preserve"`)
		}

		b.WriteString(">")
		xmlEscapeTo(b, part)
		b.WriteString("</w:t></w:r>")
	}

	b.WriteString("</w:p>")
}

func preserveSpaceNeeded(s string) bool {
	if s == "" {
		return false
	}

	return len(s) != len(strings.TrimSpace(s)) || strings.ContainsAny(s, " \t")
}

func xmlEscapeTo(b *strings.Builder, s string) {
	_ = xml.EscapeText(b, []byte(s))
}
