package document

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

const MaxRecommendedAttachmentSizeBytes = 2 * 1024 * 1024
const MaxEmbeddedAttachmentTextRunes = 48_000

var ErrUnsupportedAttachmentType = errors.New("неподдерживаемый формат вложения")
var ErrInvalidTextEncoding = errors.New("текстовый файл должен быть в UTF-8")
var ErrTextExtractionFailed = errors.New("не удалось извлечь текст из документа")
var ErrNoExtractableText = errors.New("в документе нет извлекаемого текста")

var supportedExtensions = map[string]struct{}{
	".txt":   {},
	".md":    {},
	".rst":   {},
	".log":   {},
	".json":  {},
	".yaml":  {},
	".yml":   {},
	".pdf":   {},
	".docx":  {},
	".xlsx":  {},
	".csv":   {},
	".pptx":  {},
	".html":  {},
	".htm":   {},
	".xhtml": {},
}

var binaryDocumentExtensions = map[string]struct{}{
	".pdf":  {},
	".docx": {},
	".xlsx": {},
	".csv":  {},
	".pptx": {},
}

func IsSupportedExtension(filename string) bool {
	_, ok := supportedExtensions[normalizeExt(filename)]
	return ok
}

func IsBinaryDocument(filename string) bool {
	_, ok := binaryDocumentExtensions[normalizeExt(filename)]
	return ok
}

func ValidateAttachment(filename string, content []byte) error {
	if !IsSupportedExtension(filename) {
		return ErrUnsupportedAttachmentType
	}

	if !IsBinaryDocument(filename) {
		if _, err := DecodeTextFileToUTF8(content); err != nil {
			return ErrInvalidTextEncoding
		}
	}

	return nil
}

func normalizeExt(filename string) string {
	return strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
}

func ExtractTextForRAG(filename string, content []byte) (text string, pdfPageRuneBounds []int, err error) {
	return ExtractTextForRAGContext(context.Background(), filename, content)
}

func ExtractTextForRAGContext(ctx context.Context, filename string, content []byte) (text string, pdfPageRuneBounds []int, err error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	ext := normalizeExt(filename)
	switch ext {
	case ".txt", ".md", ".log", ".rst":
		t, e := DecodeTextFileToUTF8(content)
		return t, nil, e
	case ".json":
		return extractJSONForRAG(content)
	case ".yaml", ".yml":
		return extractYAMLForRAG(content)
	case ".pdf":
		return extractPDFWithPageBounds(ctx, content)
	case ".docx":
		s, e := extractDOCX(content)
		return s, nil, e
	case ".xlsx":
		s, e := extractXLSX(content)
		return s, nil, e
	case ".csv":
		s, e := extractCSV(content)
		return s, nil, e
	case ".pptx":
		s, e := extractPPTX(content)
		return s, nil, e
	case ".html", ".htm", ".xhtml":
		text, err := DecodeTextFileToUTF8(content)
		if err != nil {
			return "", nil, err
		}
		t, e := extractHTML([]byte(text))
		return t, nil, e
	default:
		return "", nil, ErrUnsupportedAttachmentType
	}
}

func extractJSONForRAG(content []byte) (string, []int, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return "", nil, nil
	}

	dec := json.NewDecoder(bytes.NewReader(content))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t, e := DecodeTextFileToUTF8(content)
		return t, nil, e
	}

	flat := strings.TrimSpace(flattenJSONForRAG(v))
	if flat == "" {
		return "", nil, ErrNoExtractableText
	}

	return flat, nil, nil
}

func flattenJSONForRAG(v any) string {
	var parts []string
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case nil:
			return
		case string:
			if s := strings.TrimSpace(t); s != "" {
				parts = append(parts, s)
			}
		case float64, bool:
			parts = append(parts, strings.TrimSpace(fmt.Sprint(t)))
		case json.Number:
			parts = append(parts, strings.TrimSpace(t.String()))
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			for _, e := range t {
				walk(e)
			}
		default:
			parts = append(parts, strings.TrimSpace(fmt.Sprint(t)))
		}
	}

	walk(v)
	return strings.Join(parts, "\n")
}

func extractYAMLForRAG(content []byte) (string, []int, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return "", nil, nil
	}

	var n yaml.Node
	if err := yaml.Unmarshal(content, &n); err != nil {
		t, e := DecodeTextFileToUTF8(content)
		return t, nil, e
	}

	flat := strings.TrimSpace(flattenYAMLForRAG(&n))
	if flat == "" {
		return "", nil, ErrNoExtractableText
	}

	return flat, nil, nil
}

func flattenYAMLForRAG(n *yaml.Node) string {
	if n == nil {
		return ""
	}

	var parts []string
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		if node == nil {
			return
		}

		switch node.Kind {
		case yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode:
			for i := 0; i < len(node.Content); i++ {
				walk(node.Content[i])
			}
		case yaml.ScalarNode:
			s := strings.TrimSpace(node.Value)
			if s == "" || s == "null" || s == "~" {
				return
			}

			parts = append(parts, s)
		case yaml.AliasNode:
			walk(node.Alias)
		}
	}
	walk(n)
	return strings.Join(parts, "\n")
}

func ExtractText(filename string, content []byte) (string, error) {
	s, _, err := ExtractTextForRAGContext(context.Background(), filename, content)
	return s, err
}

func extractPDFWithPageBounds(ctx context.Context, content []byte) (string, []int, error) {
	tmpFile, err := os.CreateTemp("", "gen-pdf-*.pdf")
	if err != nil {
		return "", nil, fmt.Errorf("создание временного файла: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return "", nil, fmt.Errorf("запись во временный файл: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", nil, fmt.Errorf("закрытие временного файла: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	f, r, err := pdf.Open(tmpName)
	if err != nil {
		return "", nil, fmt.Errorf("открытие PDF: %w", err)
	}
	defer f.Close()

	pages := r.NumPage()
	if pages <= 0 {
		return "", []int{0, 0}, nil
	}

	fonts := make(map[string]*pdf.Font)
	var rawPerPage []string
	for i := 1; i <= pages; i++ {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}

		p := r.Page(i)
		for _, name := range p.Fonts() {
			if _, ok := fonts[name]; !ok {
				ff := p.Font(name)
				fonts[name] = &ff
			}
		}

		text, err := p.GetPlainText(fonts)
		if err != nil {
			return "", nil, fmt.Errorf("извлечение текста PDF (стр. %d): %w", i, err)
		}

		rawPerPage = append(rawPerPage, text)
	}

	normPages := make([]string, len(rawPerPage))
	for i, rp := range rawPerPage {
		normPages[i] = NormalizeExtractedText(rp)
	}

	joined := strings.Join(normPages, "\n\n")
	sep := "\n\n"
	sepR := utf8.RuneCountInString(sep)

	bounds := make([]int, len(normPages)+1)
	acc := 0
	for i, p := range normPages {
		bounds[i] = acc
		acc += utf8.RuneCountInString(p)
		if i < len(normPages)-1 {
			acc += sepR
		}
	}

	bounds[len(normPages)] = acc

	return joined, bounds, nil
}

const wordProcessingML = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

func extractDOCX(content []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("открытие DOCX (zip): %w", err)
	}

	var docXML io.ReadCloser
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			docXML, err = f.Open()
			if err != nil {
				return "", fmt.Errorf("открытие word/document.xml: %w", err)
			}
			break
		}
	}
	if docXML == nil {
		return "", fmt.Errorf("word/document.xml не найден в DOCX")
	}
	defer docXML.Close()

	raw, err := io.ReadAll(docXML)
	if err != nil {
		return "", fmt.Errorf("чтение document.xml: %w", err)
	}

	return parseWordDocumentXML(raw)
}

func parseWordDocumentXML(raw []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = false

	var paras []string
	var curPara strings.Builder
	inP, inT := false, false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("разбор document.xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWNSElem(t.Name, "p") {
				inP = true
				curPara.Reset()
			} else if inP && isWNSElem(t.Name, "t") {
				inT = true
			} else if inP && isWNSElem(t.Name, "tab") {
				curPara.WriteByte('\t')
			} else if inP && isWNSElem(t.Name, "br") {
				curPara.WriteByte('\n')
			}
		case xml.EndElement:
			if isWNSElem(t.Name, "t") {
				inT = false
			} else if isWNSElem(t.Name, "p") {
				paras = append(paras, strings.TrimRightFunc(curPara.String(), unicode.IsSpace))
				inP = false
				curPara.Reset()
			}
		case xml.CharData:
			if inT && inP {
				curPara.WriteString(string(t))
			}
		}
	}

	return strings.Join(nonEmptyLines(paras), "\n"), nil
}

func isWNSElem(name xml.Name, local string) bool {
	return name.Local == local && (name.Space == wordProcessingML || name.Space == "")
}

func nonEmptyLines(in []string) []string {
	var out []string
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}

	return out
}

func extractXLSX(content []byte) (string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("открытие XLSX: %w", err)
	}
	defer f.Close()

	var out strings.Builder
	out.WriteString("Структура: каждая строка таблицы - отдельная строка текста; столбцы в строке разделены табуляцией (TSV).\n\n")

	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return "", fmt.Errorf("чтение листа %q: %w", name, err)
		}

		if len(rows) == 0 {
			continue
		}

		out.WriteString(fmt.Sprintf("[Лист: %s]\n", name))
		for _, row := range rows {
			out.WriteString(strings.Join(row, "\t"))
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()), nil
}

func extractCSV(content []byte) (string, error) {
	text, err := DecodeTextFileToUTF8(content)
	if err != nil {
		return "", err
	}
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = detectCSVSeparator([]byte(text))
	records, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("разбор CSV: %w", err)
	}

	var out strings.Builder
	out.WriteString("Структура: каждая строка CSV - строка ниже; столбцы разделены табуляцией (после разбора исходного разделителя полей).\n\n")

	for _, row := range records {
		out.WriteString(strings.Join(row, "\t"))
		out.WriteString("\n")
	}

	return strings.TrimSuffix(out.String(), "\n"), nil
}

func TruncateExtractedText(s string, maxRunes int) (string, bool) {
	if maxRunes <= 0 || s == "" {
		return s, false
	}

	n := 0
	for i := range s {
		if n == maxRunes {
			return s[:i], true
		}
		n++
	}

	return s, false
}

func detectCSVSeparator(content []byte) rune {
	firstLine := string(content)
	if before, _, ok := bytes.Cut(content, []byte{'\n'}); ok {
		firstLine = string(before)
	}

	if strings.Contains(firstLine, ";") && !strings.Contains(firstLine, ",") {
		return ';'
	}

	return ','
}
