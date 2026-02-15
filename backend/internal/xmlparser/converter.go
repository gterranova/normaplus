package xmlparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gterranova/normaplus/backend/normattiva/document"
	"golang.org/x/net/html"
)

func FromXML(d *document.Document, xmlBytes []byte) error {
	d.Title, _ = extractTitle(xmlBytes)
	format := detectXMLFormat(xmlBytes)
	if format == "NIR" {
		return nirToDocument(d, xmlBytes)
	}
	return aknToDocument(d, xmlBytes)
}

// Global regex for detecting NIR vs AKN
// optimize by checking first few bytes?
func detectXMLFormat(xmlBytes []byte) string {
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

// extractTitle parses the XML and returns the document title.
func extractTitle(xmlBytes []byte) (string, error) {
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

func processInlineElements(root *goquery.Selection) string {
	var sb strings.Builder
	root.Contents().Each(func(_ int, selection *goquery.Selection) {
		if len(selection.Nodes) == 0 {
			return
		}
		node := selection.Nodes[0]
		if node.Type == html.ElementNode {
			tagName := goquery.NodeName(selection)
			switch tagName {
			case "ref":
				href, _ := selection.Attr("href")
				text := processInlineElements(selection)

				// Transform AKN/URN to valid Normattiva linker
				if strings.HasPrefix(href, "/akn/") {
					href = aknToUrn(href)
				}
				if strings.HasPrefix(href, "urn:nir:") {
					href = "https://www.normattiva.it/uri-res/N2Ls?" + href
				} else if strings.HasPrefix(href, "/act/") { // Sometimes relative path?
					// Fallback
					href = "https://www.normattiva.it" + href
				}

				if href != "" {
					sb.WriteString(fmt.Sprintf("[%s](%s)", text, href))
				} else {
					// Could be a link to eu regulation or directive
					sb.WriteString(text)
				}
			case "ins":
				sb.WriteString(subsAccent(processInlineElements(selection)))
			//case "authorialNote":
			//	marker, _ := s.Attr("eId")
			//	noteText := normalizeWhitespace(s.Text())
			//	if marker != "" {
			//		(*footnotes)[marker] = noteText
			//		sb.WriteString(fmt.Sprintf("[^%s]", marker))
			//	}
			case "br", "eol":
				sb.WriteString("\n")
			default:
				sb.WriteString(processInlineElements(selection))
			}
		} else if node.Type == html.TextNode {
			sb.WriteString(selection.Text())
		}
	})
	return subsAccent(sb.String())
}

func expandSelfClosingTags(xml string) string {
	// Robust regex for XML self-closing tags: <tag att="val" />
	re := regexp.MustCompile(`<([a-zA-Z0-9:]+)([^>]*?)\s*/>`)
	return re.ReplaceAllString(xml, `<$1$2></$1>`)
}

func aknToUrn(akn string) string {
	// Pattern: /akn/it/act/{type}/{authority}/{date}/{number}/...
	parts := strings.Split(akn, "/")
	if len(parts) >= 8 {
		// parts[0] = ""
		// parts[1] = "akn"
		// parts[2] = "it"
		// parts[3] = "act"
		docType := parts[4]
		authority := parts[5]
		date := parts[6]
		number := parts[7]

		// Skip EU documents
		if (docType == "regolamento" && authority == "") || authority == "eu" {
			return ""
		}

		// Sanitize inputs
		docType = strings.ReplaceAll(docType, "-", ".")
		authority = strings.ReplaceAll(authority, "-", ".")

		// If number is "0", it might be a single act (like Constitution), omit number?
		// But User example says date+number.
		// "urn:nir:stato:costituzione:1947-12-27" (no number)
		// "urn:nir:stato:legge:AAAA-MM-GG;NNN" (semicolon + number)
		urn := fmt.Sprintf("urn:nir:%s:%s:%s;%s", authority, docType, date, number)
		return strings.TrimSuffix(urn, ";0")
	}
	return akn
}
