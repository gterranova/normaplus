package converter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ToMarkdown converts Akoma Ntoso XML bytes to a well-formatted Markdown string.
func ToMarkdown(xmlBytes []byte, vigenza string) (string, error) {
	// HACK: goquery uses an HTML5 parser (net/html) which doesn't support self-closing XML tags.
	// We expand them to <tag></tag> to prevent sibling content from being swallowed into containers.
	// Multi-pass expansion handles nested self-closing tags.
	xmlStr := string(xmlBytes)
	xmlStr = expandSelfClosingTags(xmlStr)
	xmlStr = expandSelfClosingTags(xmlStr)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(xmlStr))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	footnotes := make(map[string]string)

	if vigenza != "" {
		displayDate := vigenza
		// Try to reformat YYYY-MM-DD to DD-MM-YYYY
		parts := strings.Split(vigenza, "-")
		if len(parts) == 3 {
			displayDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
		sb.WriteString(fmt.Sprintf("*Testo in vigore dal: %s*\n\n", displayDate))
	}

	// 1. Document Title
	docTitle := strings.TrimSpace(doc.Find("preface docTitle").Text())
	if docTitle != "" {
		sb.WriteString("\n\n<span id=\"preamble\"></span>\n\n")
		sb.WriteString(fmt.Sprintf("# %s\n\n", normalizeWhitespace(docTitle)))
	}

	// 2. Preamble
	doc.Find("preamble").Each(func(_ int, preamble *goquery.Selection) {
		preamble.Children().Each(func(_ int, elem *goquery.Selection) {
			tagName := goquery.NodeName(elem)
			if tagName == "formula" || tagName == "p" || tagName == "citations" {
				text := processInlineElements(elem, &footnotes)
				text = normalizeWhitespace(text)
				if text != "" {
					sb.WriteString(fmt.Sprintf("%s\n\n", text))
				}
			}
		})
	})

	// 3. Body
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		processBodyNode(s, &sb, &footnotes, 1)
	})

	// 4. Attachments
	attachments := doc.Find("attachments attachment")
	if attachments.Length() > 0 {
		sb.WriteString("## Allegati\n\n")
		attachments.Each(func(_ int, attachment *goquery.Selection) {
			processAttachment(attachment, &sb, &footnotes)
		})
	}

	// 5. Footnotes
	if len(footnotes) > 0 {
		sb.WriteString("\n---\n\n## Note\n\n")
		for marker, note := range footnotes {
			sb.WriteString(fmt.Sprintf("[^%s]: %s\n\n", marker, note))
		}
	}

	return sb.String(), nil
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
		// Fallback: try finding any capitalized heading or similar?
		// Usually Normattiva XML has docTitle.
	}
	return normalizeWhitespace(title), nil
}

func expandSelfClosingTags(xml string) string {
	// Robust regex for XML self-closing tags: <tag att="val" />
	re := regexp.MustCompile(`<([a-zA-Z0-9:]+)([^>]*?)\s*/>`)
	return re.ReplaceAllString(xml, `<$1$2></$1>`)
}

func processBodyNode(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, level int) {
	tagName := goquery.NodeName(s)

	switch tagName {
	case "chapter", "part", "title", "section":
		processStructuralContainer(s, sb, footnotes, level)
	case "article":
		// Assign to articles level 4
		processArticle(s, sb, footnotes, 4)
	default:
		s.Children().Each(func(_ int, child *goquery.Selection) {
			processBodyNode(child, sb, footnotes, level)
		})
	}
}

func processStructuralContainer(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, level int) {
	num := normalizeWhitespace(s.ChildrenFiltered("num").First().Text())

	// FIX: Stricter text extraction to avoid swallowing siblings inside unclosed tags
	headingNode := s.ChildrenFiltered("heading").First()
	heading := ""
	if headingNode.Length() > 0 {
		heading = normalizeWhitespace(extractStrictText(headingNode))
	}

	fullHeading := heading
	if num != "" && num != "-" {
		if heading != "" {
			fullHeading = fmt.Sprintf("%s %s", num, heading)
		} else {
			fullHeading = num
		}
	}

	// Inject Anchor for Deep Linking
	if eid, exists := s.Attr("eid"); exists && eid != "" {
		sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, eid) + "\n\n")
	}

	// Split complex headings (e.g. "PARTE I ... TITOLO II ...")
	headers, depthIncrease := parseComplexHeading(fullHeading, level)

	if len(headers) > 0 {
		for _, h := range headers {
			sb.WriteString(h + "\n\n")
		}
	} else {
		// Standard rendering
		lvl := level + 1
		if lvl > 6 {
			lvl = 6
		}
		prefix := strings.Repeat("#", lvl)
		if fullHeading != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n\n", prefix, fullHeading))
		}
	}

	// Calculate child level based on depth increase
	childLevel := level + 1
	if depthIncrease > 0 {
		childLevel = level + depthIncrease + 1
	}

	s.Children().Each(func(_ int, child *goquery.Selection) {
		childTag := goquery.NodeName(child)
		if childTag != "num" && childTag != "heading" {
			processBodyNode(child, sb, footnotes, childLevel-1)
		}
	})
}

func parseComplexHeading(text string, baseLevel int) ([]string, int) {
	re := regexp.MustCompile(`\b(PARTE|TITOLO|CAPO|SEZIONE)\b\s+([IVXLCDM]+|\d+)\b`)
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		re = regexp.MustCompile(`\b([A-Z\s-]+)\b`)
		matches = re.FindAllStringIndex(text, -1)
		if len(matches) == 0 {
			return nil, 0
		}
	}

	var headers []string
	currentLevel := baseLevel
	for i, match := range matches {
		start := match[0]
		end := len(text)
		if i < len(matches)-1 {
			end = matches[i+1][0]
		}

		part := text[start:end]
		part = strings.ReplaceAll(part, "- -", " ")
		part = strings.TrimSpace(part)

		lvl := currentLevel + 1
		if lvl > 6 {
			lvl = 6
		}
		headers = append(headers, fmt.Sprintf("%s %s", strings.Repeat("#", lvl), part))
		currentLevel++
	}
	return headers, len(headers)
}

func processArticle(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, level int) {

	var headingInParagraph bool

	num := normalizeWhitespace(s.ChildrenFiltered("num").First().Text())

	headingNode := s.ChildrenFiltered("heading").First()
	heading := ""

	// If the first paragraph has an eId, we skip the heading
	firstParagraphNode := s.ChildrenFiltered("paragraph").First()
	_, hasEId := firstParagraphNode.Attr("eid")

	if headingNode.Length() > 0 {
		heading = processInlineElements(headingNode, footnotes)
		heading = normalizeWhitespace(heading)
	}

	if heading == "" && !hasEId && firstParagraphNode.ChildrenFiltered("num").Length() == 0 {
		possibleHeading := processInlineElements(firstParagraphNode, footnotes)
		possibleHeading = normalizeWhitespace(possibleHeading)
		if !strings.Contains(possibleHeading, ".") {
			heading = possibleHeading
			headingInParagraph = true
		}
	}

	// Inject Anchor
	if eid, exists := s.Attr("eid"); exists && eid != "" {
		sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, eid) + "\n\n")
	}

	// Use level passed from parent. Ensure it's deep enough
	lvl := level
	if lvl < 3 {
		lvl = 3
	} // Default article is H3
	if lvl > 6 {
		lvl = 6
	}
	prefix := strings.Repeat("#", lvl)

	// User requirement: heading separated from text body by a blank line.
	header := fmt.Sprintf("%s %s", prefix, num)
	if heading != "" {
		header += fmt.Sprintf(" - %s", heading)
	}
	sb.WriteString(header + "\n\n")

	// Process content. We look at both the article's children AND children of the heading tag
	// (in case the parser incorrectly merged paragraphs into the heading tag).

	if headingNode.Length() > 0 {
		headingNode.Children().Each(func(_ int, child *goquery.Selection) {
			processArticleChild(child, sb, footnotes)
		})
	}

	s.Children().Each(func(i int, child *goquery.Selection) {
		tagName := goquery.NodeName(child)
		if tagName == "num" || tagName == "heading" {
			return
		}
		if tagName == "paragraph" && headingInParagraph {
			headingInParagraph = false
			return
		}
		processArticleChild(child, sb, footnotes)
	})
}

// processArticleChild handles individual nodes inside an article (paragraph, list, etc.)
func processArticleChild(child *goquery.Selection, sb *strings.Builder, footnotes *map[string]string) {
	tagName := goquery.NodeName(child)
	if tagName == "paragraph" || tagName == "clause" {
		processParagraph(child, sb, footnotes)
	} else if tagName == "list" {
		processList(child, sb, footnotes, 0)
		sb.WriteString("\n")
	} else if tagName == "table" {
		processTable(child, sb)
	} else if tagName == "quotedStructure" {
		processQuotedStructure(child, sb, footnotes)
	}
}

// extractStrictText returns combined text of text nodes ONLY, skipping child elements.
func extractStrictText(s *goquery.Selection) string {
	var sb strings.Builder
	s.Contents().Each(func(_ int, n *goquery.Selection) {
		if len(n.Nodes) > 0 && n.Nodes[0].Type == html.TextNode {
			sb.WriteString(n.Nodes[0].Data)
		}
	})
	return sb.String()
}

func processParagraph(para *goquery.Selection, sb *strings.Builder, footnotes *map[string]string) {
	paraNum := normalizeWhitespace(para.ChildrenFiltered("num").Text())

	// Iterate over all children to handle both wrapped <content> and direct elements (like <list>)
	para.Children().Each(func(_ int, child *goquery.Selection) {
		tagName := goquery.NodeName(child)

		if tagName == "num" {
			return
		}

		if tagName == "content" {
			child.Children().Each(func(_ int, inner *goquery.Selection) {
				processParagraphNode(inner, sb, footnotes, paraNum)
			})
		} else {
			processParagraphNode(child, sb, footnotes, paraNum)
		}
	})
}

func processParagraphNode(elem *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, paraNum string) {
	tagName := goquery.NodeName(elem)
	switch tagName {
	case "p":
		text := processInlineElements(elem, footnotes)
		text = normalizeWhitespace(text)

		if paraNum != "" {
			cleanNum := strings.TrimSuffix(paraNum, ".")
			re := regexp.MustCompile(fmt.Sprintf(`^%s\.?\s*`, regexp.QuoteMeta(cleanNum)))
			text = re.ReplaceAllString(text, "")
		}

		if text != "" {
			if strings.HasPrefix(text, "-------------") || strings.HasPrefix(text, "AGGIORNAMENTO") {
				processUpdates(elem, sb, footnotes)
			} else {
				if paraNum != "" {
					cleanNum := strings.TrimSuffix(paraNum, ".")
					// paraNum must escape trailing dot or it will be seen as a list item
					sb.WriteString(fmt.Sprintf("**%s\\.** %s\n\n", cleanNum, text))
				} else {
					sb.WriteString(fmt.Sprintf("%s\n\n", text))
				}
			}
		}
	case "list":
		if paraNum != "" {
			cleanNum := strings.TrimSuffix(paraNum, ".")
			// paraNum must escape trailing dot or it will be seen as a list item
			sb.WriteString(fmt.Sprintf("**%s\\.** ", cleanNum))
		}
		processList(elem, sb, footnotes, 0)
		sb.WriteString("\n\n")
	case "table":
		processTable(elem, sb)
	case "quotedStructure":
		processQuotedStructure(elem, sb, footnotes)
	}
}

func processList(list *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, indent int) {
	indentStr := strings.Repeat("  ", indent)
	intro := list.ChildrenFiltered("intro").Text()
	if intro != "" {
		sb.WriteString(fmt.Sprintf("%s%s\n\n", indentStr, normalizeMarkdown(normalizeWhitespace(intro))))
	}

	list.ChildrenFiltered("point, item").Each(func(i int, item *goquery.Selection) {
		num := strings.TrimSpace(item.ChildrenFiltered("num").Text())
		contentElem := item.ChildrenFiltered("content").First()
		if contentElem.Length() == 0 {
			contentElem = item
		}

		var itemTextParts []string
		contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
			t := processInlineElements(p, footnotes)
			itemTextParts = append(itemTextParts, normalizeMarkdown(normalizeWhitespace(t)))
		})
		itemText := strings.Join(itemTextParts, " ")

		if num != "" {
			// use the provided num as a list item, double line feed to separate from the previous item
			sb.WriteString(fmt.Sprintf("%s**%s** %s\n\n", indentStr, num, itemText))
		} else {
			// use a bullet point as a list item, single line feed to separate from the previous item
			sb.WriteString(fmt.Sprintf("%s- %s\n", indentStr, itemText))
		}

		item.Find("list").Each(func(_ int, subList *goquery.Selection) {
			sb.WriteString("\n")
			processList(subList, sb, footnotes, indent+1)
		})
	})
}

func processUpdates(qs *goquery.Selection, sb *strings.Builder, footnotes *map[string]string) {
	text := processInlineElements(qs, footnotes)
	lines := strings.Split(text, "\n")
	sb.WriteString("\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-------------") || strings.HasPrefix(line, "AGGIORNAMENTO") {
			sb.WriteString(fmt.Sprintf("**%s**\n\n", line))
		} else if line != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", line))
		}
	}
	sb.WriteString("\n")
}

func processQuotedStructure(qs *goquery.Selection, sb *strings.Builder, footnotes *map[string]string) {
	text := processInlineElements(qs, footnotes)
	lines := strings.Split(text, "\n")
	sb.WriteString("\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sb.WriteString(fmt.Sprintf("> %s\n", line))
		}
	}
	sb.WriteString("\n")
}

func processAttachment(att *goquery.Selection, sb *strings.Builder, footnotes *map[string]string) {

	// check if the attachment has a name attribute
	doc := att.Find("doc").First()
	heading, exists := doc.Attr("name")
	if !exists {
		heading = "Allegato"
	} else {
		// convert heading to anchor
		eid := strings.ReplaceAll(heading, " ", "-")
		eid = strings.ToLower(eid)

		// Inject Anchor
		sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, eid) + "\n\n")
	}

	sb.WriteString(fmt.Sprintf("### %s\n\n", heading))

	att.Find("mainBody").Children().Each(func(_ int, child *goquery.Selection) {
		tagName := goquery.NodeName(child)
		switch tagName {
		case "article":
			processArticle(child, sb, footnotes, 2)
		case "p", "paragraph":
			fallthrough
		default:
			text := processInlineElements(child, footnotes)
			text = strings.TrimSpace(text)
			if strings.HasPrefix(text, "-------------") || strings.HasPrefix(text, "AGGIORNAMENTO") {
				processUpdates(child, sb, footnotes)
			} else {
				text = strings.ReplaceAll(text, "\n", "\n\n")
				sb.WriteString(fmt.Sprintf("%s\n\n", text))
			}
		}
	})
}

func processInlineElements(root *goquery.Selection, footnotes *map[string]string) string {
	var sb strings.Builder
	root.Contents().Each(func(_ int, s *goquery.Selection) {
		if len(s.Nodes) == 0 {
			return
		}
		node := s.Nodes[0]
		if node.Type == html.ElementNode {
			tagName := goquery.NodeName(s)
			switch tagName {
			case "ref":
				href, _ := s.Attr("href")
				text := processInlineElements(s, footnotes)

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
				sb.WriteString(processInlineElements(s, footnotes))
			case "authorialNote":
				marker, _ := s.Attr("eId")
				noteText := normalizeWhitespace(s.Text())
				if marker != "" {
					(*footnotes)[marker] = noteText
					sb.WriteString(fmt.Sprintf("[^%s]", marker))
				}
			case "br", "eol":
				sb.WriteString("\n")
			default:
				sb.WriteString(processInlineElements(s, footnotes))
			}
		} else if node.Type == html.TextNode {
			sb.WriteString(s.Text())
		}
	})
	return normalizeMarkdown(sb.String())
}

func processTable(table *goquery.Selection, sb *strings.Builder) {
	sb.WriteString("\n")
	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		sb.WriteString("|")
		row.Find("td, th").Each(func(_ int, cell *goquery.Selection) {
			text := normalizeWhitespace(cell.Text())
			text = strings.ReplaceAll(text, "|", "\\|")
			sb.WriteString(fmt.Sprintf(" %s |", text))
		})
		sb.WriteString("\n")
		if i == 0 {
			cols := row.Find("td, th").Length()
			sb.WriteString("|")
			for k := 0; k < cols; k++ {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
		}
	})
	sb.WriteString("\n")
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
