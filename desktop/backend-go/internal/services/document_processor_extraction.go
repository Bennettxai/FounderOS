package services

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

// ============================================================================
// Text Extraction
// ============================================================================

// extractText extracts text from document content
func (p *DocumentProcessor) extractText(content []byte) (string, int, error) {
	// Check if it's plain text or markdown (UTF-8 valid and mostly printable)
	if utf8.Valid(content) && p.isProbablyText(content) {
		text := string(content)
		// Count "pages" based on ~3000 chars per page
		pageCount := (len(text) / 3000) + 1
		return text, pageCount, nil
	}

	// Try PDF extraction
	if p.isPDF(content) {
		return p.extractPDFText(content)
	}

	// Try DOCX extraction
	if p.isDOCX(content) {
		return p.extractDOCXText(content)
	}

	return "", 0, fmt.Errorf("unsupported binary format - upload as PDF, DOCX, or text")
}

// isProbablyText checks if content is likely plain text (not binary disguised as UTF-8)
func (p *DocumentProcessor) isProbablyText(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	// Check first 1000 bytes for binary indicators
	checkLen := len(content)
	if checkLen > 1000 {
		checkLen = 1000
	}

	nullCount := 0
	controlCount := 0
	for i := 0; i < checkLen; i++ {
		b := content[i]
		if b == 0 {
			nullCount++
		}
		// Count control characters (except common ones like tab, newline, carriage return)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			controlCount++
		}
	}

	// If more than 1% null bytes or 5% control chars, probably binary
	return nullCount < checkLen/100 && controlCount < checkLen/20
}

// isPDF checks if content is a PDF file
func (p *DocumentProcessor) isPDF(content []byte) bool {
	return len(content) > 4 && string(content[:4]) == "%PDF"
}

// isDOCX checks if content is a DOCX file (ZIP with specific structure)
func (p *DocumentProcessor) isDOCX(content []byte) bool {
	// DOCX files are ZIP files starting with PK signature
	if len(content) < 4 {
		return false
	}
	// Check ZIP signature
	if content[0] != 0x50 || content[1] != 0x4B {
		return false
	}
	// Try to open as ZIP and check for word/document.xml
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return false
	}
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			return true
		}
	}
	return false
}

// extractPDFText extracts text from PDF content
func (p *DocumentProcessor) extractPDFText(content []byte) (string, int, error) {
	p.logger.Info("extracting text from PDF", "size", len(content))

	// Create a temporary file for pdf library (it requires file path)
	tmpFile, err := os.CreateTemp("", "pdf_*.pdf")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		return "", 0, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Open PDF
	f, r, err := pdf.Open(tmpFile.Name())
	if err != nil {
		return "", 0, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	pageCount := r.NumPage()
	if pageCount == 0 {
		return "", 0, fmt.Errorf("PDF has no pages")
	}

	var textBuilder strings.Builder

	// Extract text from each page
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			p.logger.Warn("failed to extract text from page", "page", pageNum, "error", err)
			continue
		}

		if textBuilder.Len() > 0 {
			textBuilder.WriteString("\n\n--- Page ")
			textBuilder.WriteString(fmt.Sprintf("%d", pageNum))
			textBuilder.WriteString(" ---\n\n")
		}
		textBuilder.WriteString(text)
	}

	extractedText := strings.TrimSpace(textBuilder.String())
	if len(extractedText) == 0 {
		return "", pageCount, fmt.Errorf("no text could be extracted from PDF (may be image-based)")
	}

	p.logger.Info("PDF text extracted", "pages", pageCount, "chars", len(extractedText))
	return extractedText, pageCount, nil
}

// extractDOCXText extracts text from DOCX content
func (p *DocumentProcessor) extractDOCXText(content []byte) (string, int, error) {
	p.logger.Info("extracting text from DOCX", "size", len(content))

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", 0, fmt.Errorf("failed to open DOCX as ZIP: %w", err)
	}

	var documentXML *zip.File
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			documentXML = f
			break
		}
	}

	if documentXML == nil {
		return "", 0, fmt.Errorf("word/document.xml not found in DOCX")
	}

	rc, err := documentXML.Open()
	if err != nil {
		return "", 0, fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer rc.Close()

	xmlContent, err := io.ReadAll(rc)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read document.xml: %w", err)
	}

	// Parse DOCX XML and extract text
	extractedText := p.parseDocxXML(xmlContent)

	// Estimate page count (~3000 chars per page)
	pageCount := (len(extractedText) / 3000) + 1

	p.logger.Info("DOCX text extracted", "chars", len(extractedText), "estimated_pages", pageCount)
	return extractedText, pageCount, nil
}

// docxDocument represents the simplified DOCX XML structure
type docxDocument struct {
	Body docxBody `xml:"body"`
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text   string     `xml:"t"`
	Tab    string     `xml:"tab"`
	Break  string     `xml:"br"`
	InnerT []docxText `xml:",any"`
}

type docxText struct {
	XMLName xml.Name
	Content string `xml:",chardata"`
}

// parseDocxXML extracts plain text from DOCX XML content
func (p *DocumentProcessor) parseDocxXML(xmlContent []byte) string {
	var textBuilder strings.Builder

	// Use a simple approach: extract all text between <w:t> tags
	// This handles the complex namespace issues better than full XML parsing
	decoder := xml.NewDecoder(bytes.NewReader(xmlContent))

	var inTextElement bool
	var currentParagraph strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			p.logger.Warn("XML parsing error", "error", err)
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			localName := t.Name.Local
			if localName == "t" {
				inTextElement = true
			} else if localName == "tab" {
				currentParagraph.WriteString("\t")
			} else if localName == "br" {
				currentParagraph.WriteString("\n")
			}
		case xml.EndElement:
			localName := t.Name.Local
			if localName == "t" {
				inTextElement = false
			} else if localName == "p" {
				// End of paragraph
				if currentParagraph.Len() > 0 {
					if textBuilder.Len() > 0 {
						textBuilder.WriteString("\n")
					}
					textBuilder.WriteString(strings.TrimSpace(currentParagraph.String()))
					currentParagraph.Reset()
				}
			}
		case xml.CharData:
			if inTextElement {
				currentParagraph.Write(t)
			}
		}
	}

	// Add any remaining paragraph
	if currentParagraph.Len() > 0 {
		if textBuilder.Len() > 0 {
			textBuilder.WriteString("\n")
		}
		textBuilder.WriteString(strings.TrimSpace(currentParagraph.String()))
	}

	return strings.TrimSpace(textBuilder.String())
}
