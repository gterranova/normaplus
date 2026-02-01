package converter

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ToMarkdown converts XML bytes (AKN or NIR) to a well-formatted Markdown string.
func ToMarkdown(xmlBytes []byte, vigenza string) (string, error) {
	format := DetectXMLFormat(xmlBytes)
	if format == "NIR" {
		return NIRToMarkdown(xmlBytes, vigenza)
	}
	return AKNToMarkdown(xmlBytes, vigenza)
}

// Global regex for detecting NIR vs AKN
// optimize by checking first few bytes?
func DetectXMLFormat(xmlBytes []byte) string {
	// Simple check: NIR usually has <NIR ...> or uses DTD "NormeInRete"
	// AKN uses <akomaNtoso>
	// We can check strings.
	content := string(xmlBytes)
	if strings.Contains(content, "<akomaNtoso") {
		return "AKN"
	}
	if strings.Contains(content, "<NIR") || strings.Contains(content, "NormeInRete") {
		return "NIR"
	}
	// Fallback/Default
	return "AKN"
}

// ExtractTitle parses the XML and returns the document title.
func ExtractTitle(xmlBytes []byte) (string, error) {
	// Simple parsing just for title
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(xmlBytes)))
	if err != nil {
		return "", err
	}
	// Try standard AKN location
	title := strings.TrimSpace(doc.Find("preface docTitle").Text())

	if title == "" {
		// Try NIR location
		// <intestazione><titoloDoc>
		title = strings.TrimSpace(doc.Find("intestazione titoloDoc").Text())
	}

	if title == "" {
		// Fallback: try finding any capitalized heading or similar?
		// Usually Normattiva XML has docTitle.
	}
	return normalizeWhitespace(title), nil
}

func normalizeWhitespace(s string) string {
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func normalizeMarkdown(s string) string {
	if strings.Contains(s, "((") && strings.Contains(s, "))") {
		s = regexp.MustCompile(`\(\(\s*`).ReplaceAllString(s, "**((")
		s = regexp.MustCompile(`\s*\)\)`).ReplaceAllString(s, "))**")
	}

	return subsAccent(s)
}

func subsAccent(s string) string {
	s = strings.ReplaceAll(s, "a'", "à")
	s = strings.ReplaceAll(s, "e'", "é")
	s = strings.ReplaceAll(s, "i'", "ì")
	s = strings.ReplaceAll(s, "o'", "ò")
	s = strings.ReplaceAll(s, "u'", "ù")
	s = strings.ReplaceAll(s, "E'", "È")
	s = strings.ReplaceAll(s, "pò", "po'")
	//s = strings.ReplaceAll(s, "Pò", "Po'")
	s = strings.ReplaceAll(s, " é ", " è ")
	if strings.HasSuffix(s, " é") {
		s = s[:len(s)-2] + "è"
	}
	if strings.HasPrefix(s, "é ") {
		s = "è" + s[2:]
	}
	return s
}
