package xmlparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gterranova/normaplus/backend/normattiva/document"
	"golang.org/x/net/html"
)

func nirToDocument(d *document.Document, xmlBytes []byte) error {
	// HACK: goquery uses an HTML5 parser (net/html) which doesn't support self-closing XML tags.
	xmlStr := string(xmlBytes)
	xmlStr = expandSelfClosingTags(xmlStr)
	xmlStr = expandSelfClosingTags(xmlStr)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(xmlStr))
	if err != nil {
		return err
	}

	// 1. Document Title
	docTitle := strings.TrimSpace(doc.Find("intestazione titoloDoc").Text())
	if docTitle != "" {
		d.Title = subsAccent(normalizeWhitespace(docTitle))
	}

	// 2. Preamble
	preambleSection := document.NewDocumentSection("preamble", "", d)
	doc.Find("formulainiziale").Each(func(_ int, preamble *goquery.Selection) {
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
	if len(preambleSection.Content) > 0 {
		d.AddSection(preambleSection)
	}

	// 3. Body
	doc.Find("body").Children().Each(func(i int, selection *goquery.Selection) {
		tagName := goquery.NodeName(selection)
		section := document.NewDocumentSection(tagName, "", d)
		processNIRBodyNode(&section, selection, 0)
		d.AddSection(section)
	})

	// 4. Attachments
	attachments := doc.Find("attachments attachment")
	if attachments.Length() > 0 {
		attachmentsSection := document.NewDocumentSection("attachments", "Allegati", d)
		processNIRAttachment(&attachmentsSection, attachments)
		d.AddSection(attachmentsSection)
	}
	return nil
}

// NIR to json helpers
func processNIRContent(s *document.DocumentSection, selection *goquery.Selection) string {
	// 1. Text Node
	if len(selection.Nodes) > 0 && selection.Nodes[0].Type == html.TextNode {
		t := strings.TrimSpace(selection.Text())
		t = strings.ReplaceAll(t, "<![CDATA[", "> ")
		t = strings.ReplaceAll(t, "]]>", "")
		return t
	}

	// 2. Element Node - Check Tag
	tagName := goquery.NodeName(selection)
	cleanTag := strings.TrimPrefix(tagName, "h:")

	switch cleanTag {
	case "p", "div":
		// Indentation handling
		style, _ := selection.Attr("h:style")
		if style == "" {
			style, _ = selection.Attr("style")
		}

		prefix := ""
		if strings.Contains(style, "padding-left") {
			parts := strings.Split(style, ";")
			for _, p := range parts {
				if strings.Contains(p, "padding-left") {
					val := strings.TrimSpace(strings.Split(p, ":")[1])
					val = strings.ReplaceAll(val, "px", "")
					val = strings.TrimSpace(val)
					switch val {
					case "4":
						prefix = "> "
					case "6":
						prefix = "> >  "
					case "8":
						prefix = "> > >  "
					default:
						// ignore
					}
				}
			}
		}

		content := processNIRInner(s, selection)
		if prefix != "" {
			return prefix + content
		}
		return content

	case "br":
		return "\n"

	case "table":
		return processNIRTable(selection)

	case "a":
		href, _ := selection.Attr("href")
		text := processNIRInner(s, selection)
		if href != "" {
			return fmt.Sprintf("[%s](%s)", text, href)
		}
		return text

	case "span", "b", "strong", "i", "em", "testata", "denAnnesso", "titAnnesso":
		// Formatting could be added here, for now pass through
		return processNIRInner(s, selection)

	case "ndr":
		text := selection.AttrOr("value", "")
		if text == "" {
			text = processNIRInner(s, selection)
		}
		return text

	default:
		return processNIRInner(s, selection)
	}
}

func processNIRInner(s *document.DocumentSection, selection *goquery.Selection) string {
	var sb strings.Builder
	selection.Contents().Each(func(_ int, child *goquery.Selection) {
		t := processNIRContent(s, child)
		sb.WriteString(t)
		sb.WriteString("\n\n")
	})
	reNum := regexp.MustCompile(`^([\(]*\d+[\.\d]*[a-z\-]*)\.\s*`)
	text := subsAccent(normalizeWhitespace(sb.String()))
	text = reNum.ReplaceAllString(text, "$1\\. ")

	return text
}

func processNIRBodyNode(s *document.DocumentSection, selection *goquery.Selection, level int) {
	tagName := goquery.NodeName(selection)
	section := document.NewDocumentSection(tagName, "", s.Root)

	switch tagName {
	case "libro", "parte", "titolo", "capo", "sezione":
		processNIRContainer(&section, selection, level+1)
		s.AddSection(section)
	case "articolo":
		processNIRArticle(s, selection, 4)
	default:
		selection.Children().Each(func(_ int, child *goquery.Selection) {
			processNIRBodyNode(s, child, level)
		})
	}
}

func processNIRContainer(s *document.DocumentSection, selection *goquery.Selection, level int) {
	tagName := goquery.NodeName(selection)
	num := normalizeWhitespace(selection.ChildrenFiltered("num").First().Text())
	rubrica := normalizeWhitespace(selection.ChildrenFiltered("rubrica").First().Text())

	fullHeading := num
	if rubrica != "" {
		if fullHeading != "" {
			fullHeading += " - " + rubrica
		} else {
			fullHeading = rubrica
		}
	}
	s.Title = subsAccent(normalizeWhitespace(fullHeading))

	// Inject Anchor
	if id, exists := selection.Attr("id"); exists && id != "" {
		s.ID = fmt.Sprintf("%s_%s", tagName, id)
	}

	selection.Children().Each(func(_ int, child *goquery.Selection) {
		childTag := goquery.NodeName(child)
		if childTag != "num" && childTag != "rubrica" {
			section := document.NewDocumentSection(childTag, "", s.Root)
			processNIRBodyNode(&section, child, level+1)
			s.AddSection(section)
		}
	})
}

func processNIRArticle(s *document.DocumentSection, selection *goquery.Selection, level int) {
	num := strings.TrimSuffix(normalizeWhitespace(selection.ChildrenFiltered("num").First().Text()), ".")
	rubrica := subsAccent(normalizeWhitespace(selection.ChildrenFiltered("rubrica").First().Text()))

	// Special handling for Article 1: Preamble might be buried inside
	preambleNodes, cleanArticleNodes, foundInternalHeader := detectAndSplitPreamble(selection, num)

	var skipNodes int

	// If we found a split, we render the preamble FIRST
	if len(preambleNodes) > 0 {
		preambleSection := document.NewDocumentSection("preamble", "", s.Root)
		for _, node := range preambleNodes {
			// We process these nodes as body content (level 0?)
			// Usually these are <p> tags from the first comma
			t := processNIRContent(s, node)
			if strings.TrimSpace(t) != "" {
				preambleSection.AddContent(t)
			}
		}
		if len(preambleSection.Content) > 0 {
			s.Root.Sections = append([]document.DocumentSection{preambleSection}, s.Root.Sections...)
		}
	}

	// If we found an internal header, we might need to re-evaluate Num/Rubrica
	// But usually we trust the internal "Art. 1" text we found to confirm the split.
	// The Rubrica might be the *next* node after "Art. 1".
	if foundInternalHeader {
		// Attempt to extract rubrica from cleanArticleNodes (post-split)
		// The first node might be "Art. 1", second might be "Rubrica"
		// detectAndSplitPreamble already consumes the "Art. 1" node usually?
		// Let's rely on standard flow or re-extract?

		// If rubrica was empty before, try to find it in the new clean nodes
		if rubrica == "" && len(cleanArticleNodes) > 0 {
			nodesChecked := 0
			for i, node := range cleanArticleNodes {
				if nodesChecked > 5 {
					break
				} // Check more nodes, but stop eventually
				t := normalizeWhitespace(node.Text())
				if t == "" {
					continue
				} // Skip whitespace nodes

				nodesChecked++
				style, _ := node.Attr("h:style")

				// Heuristic: Centered or short text
				// Stop if we hit something that looks like the start of body (e.g. starts with "1.")
				possibleFirstPara := regexp.MustCompile(`^[\(]+\s*1\.?`).MatchString(t)
				if nodesChecked > 0 && possibleFirstPara {
					// Likely hit body
					break
				}

				if strings.Contains(style, "center") || (len(t) < 200 && !possibleFirstPara) {
					rubrica = t
					cleanArticleNodes = append(cleanArticleNodes[:i], cleanArticleNodes[i+1:]...)
					break
				}
			}
		}
	} else if rubrica == "" {
		// Fallback: If Rubrica is missing AND no internal header split happened
		firstComma := selection.ChildrenFiltered("comma").First()
		if firstComma.Length() > 0 {
			rubrica, skipNodes = extractFallbackRubrica(firstComma, num)
		}
	}

	rubrica = normalizeWhitespace(rubrica)

	header := num
	if rubrica != "" {
		num = regexp.MustCompile(`^(Art\.\s*\d+)[\s\-]([a-z]+)`).ReplaceAllString(num, "$1-$2")
		rubrica = strings.TrimPrefix(rubrica, num)
		rubrica = normalizeWhitespace(rubrica)
		header = fmt.Sprintf("%s - %s", num, rubrica)
	}

	s.ID = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(num, ".", "_"), " ", ""))
	s.Title = header

	// If we have cleanArticleNodes (from split), iterate them
	// Note: cleanArticleNodes are purely the *contents* of the first comma (and maybe others?)
	// Actually, `detectAndSplitPreamble` scans the first comma.
	// We need to handle the rest of the comma/article children.

	if foundInternalHeader {
		// We manually found the split in the first comma.
		// We assume `cleanArticleNodes` contains the valid *content nodes* of the *first* comma.
		// We still need to process subsequent commas if any.

		// 1. Render the clean nodes of the first comma (simulating processNIRComma)
		// We need to verify if there's a Num for the first comma?
		// Usually if Preamble is mixed in, it's just one big blob.
		// Let's render cleanArticleNodes as the body of Art 1 (or Comma 1).

		for _, node := range cleanArticleNodes {
			t := processNIRContent(s, node)
			if strings.TrimSpace(t) != "" {
				s.AddContent(t)
			}
		}

		// 2. Process REMAINING commas (after the first one)
		selection.Children().Each(func(i int, child *goquery.Selection) {
			tagName := goquery.NodeName(child)
			if tagName == "num" || tagName == "rubrica" {
				return
			}

			// Skip the first comma as we handled it via split
			// Actually index i in contents vs filtered "comma"?
			// s.Children() includes everything.
			// We need to be careful identifying the "first comma".
			// Check against the one we split?
			// Simplification: assume first "comma" tag is the one we processed.
			isFirstComma := false
			if tagName == "comma" {
				// Check if it's the first *comma*
				if child.PrevFiltered("comma").Length() == 0 {
					isFirstComma = true
				}
			}

			if isFirstComma {
				return
			}

			processNIRArticleChild(s, child, 0, num, rubrica)
		})

	} else {
		// Standard path
		selection.Children().Each(func(i int, child *goquery.Selection) {
			tagName := goquery.NodeName(child)
			if tagName == "num" || tagName == "rubrica" {
				return
			}

			// If this is the first comma and we found a fallback rubrica, tell it to skip nodes
			skip := 0
			if i == 0 && skipNodes > 0 && tagName == "comma" {
				skip = skipNodes
			}

			processNIRArticleChild(s, child, skip, num, rubrica)
			// Reset skip after first use (though unlikely to apply again)
			if skip > 0 {
				skipNodes = 0
			}
		})
	}
}

// detectAndSplitPreamble checks if Article 1 actually contains the Preamble.
// Returns:
// - Preamble Nodes (before split)
// - Article Nodes (after split)
// - Boolean (true if split occurred)
func detectAndSplitPreamble(selection *goquery.Selection, articleNum string) ([]*goquery.Selection, []*goquery.Selection, bool) {
	if !strings.Contains(articleNum, "1") {
		return nil, nil, false // Only check Art 1 (or close to it)
	}

	firstComma := selection.ChildrenFiltered("comma").First()
	if firstComma.Length() == 0 {
		return nil, nil, false
	}

	// Check inside <corpo> if exists
	container := firstComma
	if corpo := firstComma.ChildrenFiltered("corpo"); corpo.Length() > 0 {
		container = corpo
	}

	var preamble []*goquery.Selection
	var article []*goquery.Selection
	splitFound := false

	// Iterate ALL contents (text nodes + elements)
	container.Contents().Each(func(i int, node *goquery.Selection) {
		if splitFound {
			article = append(article, node)
			return
		}

		// Check for split marker
		tagName := goquery.NodeName(node)
		if strings.Contains(tagName, "p") {
			text := normalizeWhitespace(node.Text())
			style, _ := node.Attr("h:style")
			cleanNum := strings.TrimRight(articleNum, ".")
			cleanText := strings.TrimRight(text, ".")

			// Check for "Art. 1" centered
			if strings.EqualFold(cleanText, cleanNum) {
				if strings.Contains(style, "center") || strings.Contains(style, "zh2") { // zh2 is common NIR class for headers
					splitFound = true
					// We consume this node as the header separator, don't add to article or preamble
					return
				}
			}
		}

		preamble = append(preamble, node)
	})

	if splitFound {
		return preamble, article, true
	}
	return nil, nil, false
}

func processNIRArticleChild(s *document.DocumentSection, selection *goquery.Selection, skipNodes int, articleNum string, rubrica string) {
	tagName := goquery.NodeName(selection)

	if tagName == "comma" {
		processNIRComma(s, selection, skipNodes, articleNum, rubrica)
	} else {
		// e.g. direct text or other tags?
		// Usually articles have commas. If strictly <corpo>...
		text := processNIRContent(s, selection)
		if text != "" {
			s.AddContent(text)
		}
	}
}

func processNIRComma(s *document.DocumentSection, selection *goquery.Selection, skipNodes int, articleNum string, articleRubrica string) {
	//num := normalizeWhitespace(selection.ChildrenFiltered("num").First().Text())

	var sb strings.Builder

	// The comma might have <corpo>
	container := selection
	if corpo := selection.ChildrenFiltered("corpo"); corpo.Length() > 0 {
		container = corpo
	}

	container.Contents().Each(func(i int, child *goquery.Selection) {
		// skip logic for headers consumed?
		if skipNodes > 0 && i < skipNodes {
			return
		}

		t := processNIRContent(s, child)
		if strings.TrimSpace(t) != "" {
			t = strings.TrimPrefix(t, articleNum)
			t = strings.TrimPrefix(t, articleRubrica)
			sb.WriteString(t)
			sb.WriteString("\n\n")
		}
	})

	text := sb.String()

	sb.Reset()

	currentComma := ""
	currentNum := 0

	extractNumberFromComma := func(comma string) int {
		// extract the number from the comma
		re := regexp.MustCompile(`^\d+`)
		comma = strings.TrimPrefix(comma, "(")
		comma = strings.TrimPrefix(comma, "(")
		match := re.FindString(comma)
		if match == "" {
			return 0
		}
		num, err := strconv.Atoi(match)
		if err != nil {
			return 0
		}
		return num
	}

	newCommaRe := regexp.MustCompile(`^[\(\s]*\d+[a-z-]*\\?\.[\s\)\)]+`)
	for _, line := range strings.Split(text, "\n\n") {
		line = normalizeWhitespace(line)
		if line != "" {
			possibleNewComma := newCommaRe.FindString(line)
			num := extractNumberFromComma(possibleNewComma)
			// if possibleNewComma is found, it means it's a new comma
			if len(possibleNewComma) > 0 && num >= currentNum {
				currentNum = num
				if possibleNewComma != currentComma && currentComma != "" {
					s.AddContent(subsAccent(strings.TrimSpace(sb.String())))
					sb.Reset()
				}
				currentComma = possibleNewComma
			}
			sb.WriteString(line + "\n\n")
		}
	}

	remaining := sb.String()
	if remaining != "" {
		s.AddContent(subsAccent(strings.TrimSpace(remaining)))
	}

	/*
		if num != "" {
			// Ensure num has dot
			if !strings.HasSuffix(num, ".") {
				num += "\\."
			}
			s.AddContent(fmt.Sprintf("%s %s", num, text))
		} else {
			// If no num, just text (common for single-comma articles)
			s.AddContent(text)
		}
	*/
}

// extractFallbackRubrica attempts to find a title hidden in the body of the first comma.
// Returns the extracted rubrica and the number of paragraph nodes to skip.
func extractFallbackRubrica(comma *goquery.Selection, articleNum string) (string, int) {
	// Look inside <corpo> or comma children for <p> tags
	corpo := comma.ChildrenFiltered("corpo")
	container := comma
	if corpo.Length() > 0 {
		container = corpo
	}

	var nodesToSkip int
	var foundRubrica []string

	// We use Contents() to match the iteration logic in processNIRComma
	// This ensures indices align (including text nodes)
	container.Contents().EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i > 15 {
			return false
		} // Look a bit deeper as whitespace text nodes count

		// If it's a text node (mostly whitespace), we check if we should keep it or if it's part of the header area
		if len(s.Nodes) > 0 && s.Nodes[0].Type == html.TextNode {
			// If we are still searching, we might skip leading whitespace.
			// But we need to increment the counter if we eventually find the header.
			// So we just continue. If we find the header at index 5, we skip 0-5.
			return true
		}

		tagName := goquery.NodeName(s)

		if strings.Contains(tagName, "p") { // h:p or p
			text := normalizeWhitespace(s.Text())
			style, _ := s.Attr("h:style")

			cleanNum := strings.TrimRight(articleNum, ".")
			cleanText := strings.TrimRight(text, ".")

			// Match "Art. 2"
			if strings.EqualFold(cleanText, cleanNum) {
				nodesToSkip = i + 1
				return true
			}

			// Match Rubrica (centered or if we already found Art. 2)
			// Also checking if text length is reasonable for a title (not a full paragraph)
			// AND explicitly excluding text that looks like the start of the body (e.g. "1.")
			possibleFirstPara := regexp.MustCompile(`^[\(]*\s*1\.?`).MatchString(text)

			if strings.Contains(style, "center") || (nodesToSkip > 0 && len(text) < 200 && !possibleFirstPara) {
				foundRubrica = append(foundRubrica, text)
				nodesToSkip = i + 1
				return true
			}

			// If we hit a non-centered paragraph and haven't found anything or finished finding headers
			// Stop.
			return false
		}

		// Skip br tags in the header area
		if strings.Contains(tagName, "br") {
			// If we are in the middle of finding headers, this is skippable.
			// But if we haven't found anything yet, it's just a br.
			// We can optimistically include it in skip if we find a header later.
			return true
		}

		// Any other tag stops the search
		return false
	})

	return strings.Join(foundRubrica, " "), nodesToSkip
}

func processNIRAttachment(s *document.DocumentSection, selection *goquery.Selection) {
	// <annesso id="...">
	//   <testata>
	//     <denAnnesso>Allegato A</denAnnesso>
	//     <titAnnesso>...</titAnnesso>
	//   </testata>
	//   <rifesterno link="..."/> OR Content
	// </annesso>

	den := normalizeWhitespace(selection.Find("testata denAnnesso").Text())
	tit := normalizeWhitespace(selection.Find("testata titAnnesso").Text())

	heading := den
	if tit != "" {
		if heading != "" {
			heading += " - " + tit
		} else {
			heading = tit
		}
	}

	if id, exists := selection.Attr("id"); exists && id != "" {
		s.AddContent(fmt.Sprintf(`<span id="%s"></span>`, id))
	}

	attachmentSection := document.NewDocumentSection("attachment", heading, s.Root)

	// Content could be linking to external PDF or embedded
	// In the provided example, we don't see attachments structure deep dive.
	// Assuming generic content processing:
	selection.Children().Each(func(_ int, child *goquery.Selection) {
		if goquery.NodeName(child) != "testata" && goquery.NodeName(child) != "meta" {
			// e.g. if it has <rifesterno>
			if goquery.NodeName(child) == "rifesterno" {
				link, _ := child.Attr("xlink:href") // usually URN
				attachmentSection.AddContent(fmt.Sprintf("[Vedi Allegato](%s)\n\n", link))
			} else {
				// Process as body node
				processNIRBodyNode(s, child, 4)
			}
		}
	})

	s.AddSection(attachmentSection)
}

func processNIRTable(child *goquery.Selection) string {
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

	return sb.String()
}
