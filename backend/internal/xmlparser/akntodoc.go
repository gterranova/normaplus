package xmlparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"github.com/gterranova/normaplus/backend/normattiva/document"
)

func aknToDocument(d *document.Document, xmlBytes []byte) error {
	// HACK: goquery uses an HTML5 parser (net/html) which doesn't support self-closing XML tags.
	// We expand them to <tag></tag> to prevent sibling content from being swallowed into containers.
	// Multi-pass expansion handles nested self-closing tags.
	xmlStr := string(xmlBytes)
	xmlStr = expandSelfClosingTags(xmlStr)
	xmlStr = expandSelfClosingTags(xmlStr)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(xmlStr))
	if err != nil {
		return err
	}

	// 1. Document Title
	docTitle := strings.TrimSpace(doc.Find("preface docTitle").Text())
	if docTitle != "" {
		d.Title = normalizeWhitespace(docTitle)
	}

	// 2. Preamble
	preambleSection := document.NewDocumentSection("preamble", "", d)
	doc.Find("preamble").Each(func(_ int, preamble *goquery.Selection) {
		preamble.Children().Each(func(_ int, elem *goquery.Selection) {
			tagName := goquery.NodeName(elem)
			if tagName == "formula" || tagName == "p" || tagName == "citations" {
				text := processInlineElements(elem)
				text = subsAccent(normalizeWhitespace(text))
				if text != "" {
					preambleSection.AddContent(text)
				}
			}
		})
	})
	d.AddSection(preambleSection)

	// 3. Body
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		tagName := goquery.NodeName(s)
		section := document.NewDocumentSection(tagName, "", d)
		processBodyNode(&section, s)
		d.AddSection(section)
	})

	// 4. Attachments
	attachments := doc.Find("attachments attachment")
	if attachments.Length() > 0 {
		attachmentsSection := document.NewDocumentSection("attachments", "Allegati", d)
		processAttachmentNode(&attachmentsSection, attachments)
		d.AddSection(attachmentsSection)
	}

	return nil
}

func processBodyNode(s *document.DocumentSection, selection *goquery.Selection) {
	tagName := goquery.NodeName(selection)

	switch tagName {
	case "chapter", "part", "title", "section":
		section := document.NewDocumentSection(tagName, "", s.Root)
		processStructuralContainer(&section, selection)
		s.AddSection(section)
	case "article":
		section := document.NewDocumentSection("article", "", s.Root)
		// Assign to articles level 4
		processArticle(&section, selection)
		s.AddSection(section)
	default:
		selection.Children().Each(func(_ int, child *goquery.Selection) {
			processBodyNode(s, child)
		})
	}
}

func processStructuralContainer(s *document.DocumentSection, selection *goquery.Selection) {
	num := normalizeWhitespace(selection.ChildrenFiltered("num").First().Text())

	// FIX: Stricter text extraction to avoid swallowing siblings inside unclosed tags
	headingNode := selection.ChildrenFiltered("heading").First()
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
	s.Title = subsAccent(normalizeWhitespace(fullHeading))

	// Inject Anchor for Deep Linking
	if eid, exists := selection.Attr("eid"); exists && eid != "" {
		s.ID = eid
	}

	/*
		// Split complex headings (e.g. "PARTE I ... TITOLO II ...")
		deepSection, _ := s.parseComplexHeading(fullHeading)

		selection.Children().Each(func(_ int, child *goquery.Selection) {
			childTag := goquery.NodeName(child)
			if childTag != "num" && childTag != "heading" {
				deepSection.processBodyNode(child)
			}
		})
	*/
	selection.Children().Each(func(_ int, child *goquery.Selection) {
		childTag := goquery.NodeName(child)
		if childTag != "num" && childTag != "heading" {
			processBodyNode(s, child)
		}
	})
}

func processArticle(s *document.DocumentSection, selection *goquery.Selection) {

	var headingInParagraph bool

	num := normalizeWhitespace(selection.ChildrenFiltered("num").First().Text())

	headingNode := selection.ChildrenFiltered("heading").First()
	heading := ""

	// If the first paragraph has an eId, we skip the heading
	firstParagraphNode := selection.ChildrenFiltered("paragraph").First()
	_, hasEId := firstParagraphNode.Attr("eid")

	if headingNode.Length() > 0 {
		heading = processInlineElements(headingNode)
		heading = subsAccent(normalizeWhitespace(heading))
	}

	if heading == "" && !hasEId && firstParagraphNode.ChildrenFiltered("num").Length() == 0 {
		possibleHeading := processInlineElements(firstParagraphNode)
		possibleHeading = subsAccent(normalizeWhitespace(possibleHeading))
		if !strings.Contains(possibleHeading, ".") {
			heading = possibleHeading
			headingInParagraph = true
		}
	}

	// Inject Anchor
	if eid, exists := selection.Attr("eid"); exists && eid != "" {
		s.ID = eid
	}

	// User requirement: heading separated from text body by a blank line.
	header := num
	if heading != "" {
		cleanNum := strings.TrimSuffix(num, ".")
		header = fmt.Sprintf("%s - %s", cleanNum, heading)
	}
	s.Title = header

	// Process content. We look at both the article's children AND children of the heading tag
	// (in case the parser incorrectly merged paragraphs into the heading tag).

	if headingNode.Length() > 0 {
		headingNode.Children().Each(func(_ int, child *goquery.Selection) {
			processArticleChild(s, child)
		})
	}

	selection.Children().Each(func(i int, child *goquery.Selection) {
		tagName := goquery.NodeName(child)
		if tagName == "num" || tagName == "heading" {
			return
		}
		if tagName == "paragraph" && headingInParagraph {
			headingInParagraph = false
			return
		}
		processArticleChild(s, child)
	})
}

// processArticleChild handles individual nodes inside an article (paragraph, list, etc.)
func processArticleChild(s *document.DocumentSection, child *goquery.Selection) {
	tagName := goquery.NodeName(child)
	switch tagName {
	case "paragraph", "clause":
		processParagraph(s, child)
	case "list":
		s.AddContent(processList(child, 0))
	case "table":
		s.AddContent(processTable(child))
	case "quotedStructure":
		s.AddContent(processQuotedStructure(child))
	}
}

func processParagraph(s *document.DocumentSection, child *goquery.Selection) {
	paraNum := normalizeWhitespace(child.ChildrenFiltered("num").Text())

	// Iterate over all children to handle both wrapped <content> and direct elements (like <list>)
	child.Children().Each(func(_ int, child *goquery.Selection) {
		tagName := goquery.NodeName(child)

		if tagName == "num" {
			return
		}

		if tagName == "content" {
			child.Children().Each(func(_ int, inner *goquery.Selection) {
				processParagraphNode(s, inner, paraNum)
			})
		} else {
			processParagraphNode(s, child, paraNum)
		}
	})
}

func processParagraphNode(s *document.DocumentSection, child *goquery.Selection, paraNum string) {
	tagName := goquery.NodeName(child)

	var text string
	switch tagName {
	case "p":
		text = processInlineElements(child)
		//text = normalizeWhitespace(text)

		if paraNum != "" {
			cleanNum := strings.TrimSuffix(paraNum, ".")
			re := regexp.MustCompile(fmt.Sprintf(`^[\(\s]*%s\.?\s*`, regexp.QuoteMeta(cleanNum)))
			text = re.ReplaceAllString(text, "")
		}

		lines := strings.Split(strings.TrimSpace(text), "\n")
		if len(lines) > 1 {
			_ = normalizeWhitespace(lines[0])
		}
		for _, line := range lines {
			line = subsAccent(normalizeWhitespace(line))
			if strings.HasPrefix(line, "((") && strings.HasSuffix(line, "))") {
				possibleNewComma := regexp.MustCompile(`^[\(\s]*\d+[a-z-]*\.?`).FindString(line)
				if len(possibleNewComma) > 0 {
					if len(line) > len(possibleNewComma)+2 {
						paraNum = possibleNewComma[2:]
						line = "((" + strings.TrimSpace(line[len(possibleNewComma):])
					} else {
						// it is a note
					}
				}
			}
			if line != "" {
				if line == "((" || line == "))" {
					continue
				}
				if strings.HasPrefix(line, "---") {
					processUpdates(s, child)
					break
				} else {
					if paraNum != "" {
						cleanNum := strings.TrimSuffix(paraNum, ".")
						// paraNum must escape trailing dot or it will be seen as a list item
						s.AddContent(fmt.Sprintf("%s\\. %s", cleanNum, line))
						paraNum = ""
					} else {
						s.AddContent(line)
					}
				}
			}
		}
		return

	case "list":
		text = processList(child, 0)
		if paraNum != "" {
			cleanNum := strings.TrimSuffix(paraNum, ".")
			// paraNum must escape trailing dot or it will be seen as a list item
			text = fmt.Sprintf("%s\\. %s", cleanNum, text)
		}
	case "table":
		text = processTable(child)
	case "quotedStructure":
		text = processQuotedStructure(child)
	}

	if strings.TrimSpace(text) != "" {
		s.AddContent(text)
	}
}

func processList(child *goquery.Selection, indent int) string {

	var sb strings.Builder

	indentStr := strings.Repeat("  ", indent)
	intro := child.ChildrenFiltered("intro").Text()
	if intro != "" {
		sb.WriteString(fmt.Sprintf("%s%s\n\n", indentStr, normalizeWhitespace(intro)))
	}

	child.ChildrenFiltered("point, item").Each(func(i int, item *goquery.Selection) {
		num := strings.TrimSpace(item.ChildrenFiltered("num").Text())
		contentElem := item.ChildrenFiltered("content").First()
		if contentElem.Length() == 0 {
			contentElem = item
		}

		var itemTextParts []string
		contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
			t := processInlineElements(p)
			itemTextParts = append(itemTextParts, normalizeWhitespace(t))
		})
		itemText := strings.Join(itemTextParts, " ")

		if num != "" {
			// use the provided num as a list item, double line feed to separate from the previous item
			sb.WriteString(fmt.Sprintf("%s%s %s\n\n", indentStr, num, itemText))
		} else {
			// use a bullet point as a list item, single line feed to separate from the previous item
			sb.WriteString(fmt.Sprintf("%s- %s\n", indentStr, itemText))
		}

		item.Find("list").Each(func(_ int, subList *goquery.Selection) {
			sb.WriteString(processList(subList, indent+1))
		})
	})

	return subsAccent(sb.String())
}

func processTable(child *goquery.Selection) string {
	var sb strings.Builder

	child.Find("tr").Each(func(i int, row *goquery.Selection) {
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

	return subsAccent(sb.String())
}

func processQuotedStructure(child *goquery.Selection) string {
	var sb strings.Builder

	text := processInlineElements(child)
	lines := strings.Split(text, "\n")
	sb.WriteString("\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sb.WriteString(fmt.Sprintf("> %s\n", line))
		}
	}
	sb.WriteString("\n")

	return subsAccent(sb.String())
}

func processUpdates(s *document.DocumentSection, child *goquery.Selection) {
	var sb strings.Builder

	text := processInlineElements(child)
	lines := strings.Split(text, "\n")

	isUpdate := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !isUpdate && strings.HasPrefix(line, "---") {
			isUpdate = true
			continue
		} else if isUpdate && line != "" {
			sb.WriteString(fmt.Sprintf("> %s\n> \n", line))
		}
	}
	s.AddContent(subsAccent(sb.String()))
}

func processAttachmentNode(s *document.DocumentSection, selection *goquery.Selection) {

	selection.Children().Each(func(_ int, doc *goquery.Selection) {
		// check if the attachment has a name attribute
		attachmentSection := document.NewDocumentSection("attachments", "", s.Root)
		heading, exists := doc.Attr("name")
		if !exists {
			heading = "Allegato"
		} else {
			// convert heading to anchor
			eid := strings.ReplaceAll(heading, " ", "-")
			eid = strings.ToLower(eid)

			// Inject Anchor
			attachmentSection.ID = eid
		}

		attachmentSection.Title = subsAccent(heading)

		doc.Find("mainBody").Children().Each(func(_ int, child *goquery.Selection) {
			tagName := goquery.NodeName(child)
			switch tagName {
			case "article":
				processArticle(&attachmentSection, child)
			case "p", "paragraph":
				processParagraph(&attachmentSection, child)
			default:
				text := processInlineElements(child)
				text = strings.TrimSpace(subsAccent(text))
				if strings.HasPrefix(text, "-------------") || strings.HasPrefix(text, "AGGIORNAMENTO") {
					processUpdates(&attachmentSection, child)
				} else {
					text = strings.ReplaceAll(text, "\n", "\n\n")
					attachmentSection.AddContent(text)
				}
			}
		})

		s.AddSection(attachmentSection)
	})
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

/*
func processInlineElements(root *goquery.Selection) string {
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
				text := processInlineElements(s)

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
				sb.WriteString(processInlineElements(s))
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
				sb.WriteString(processInlineElements(s))
			}
		} else if node.Type == html.TextNode {
			sb.WriteString(s.Text())
		}
	})
	return normalizeMarkdown(sb.String())
}
*/

/*
func parseComplexHeading(s *document.DocumentSection, text string) (*document.DocumentSection, bool) {
	re := regexp.MustCompile(`\b(PARTE|TITOLO|CAPO|SEZIONE)\s+([IVXLCDM]+|\d+)`)
	matches := re.FindAllStringIndex(strings.ToUpper(text), -1)
	if len(matches) == 0 {
		//re = regexp.MustCompile(`\b([A-Z\s-]+)\b`)
		//matches = re.FindAllStringIndex(text, -1)
		//if len(matches) == 0 {
		return s, false
		//}
	}

	var currentSection *DocumentSection
	for i, match := range matches {
		if currentSection == nil {
			currentSection = s
		}
		start := match[0]
		end := len(text)
		if i < len(matches)-1 {
			end = matches[i+1][0]
		}

		if i > 0 {
			part := text[start:end]
			part = strings.ReplaceAll(part, "- -", " ")
			part = strings.TrimSpace(part)

			tipo := strings.ToLower(strings.Fields(part)[0])
			deepSection := NewDocumentSection(tipo, part, s.Root)
			currentSection.AddSection(deepSection)
			currentSection = &deepSection
		}
	}
	return currentSection, true
}
*/
