package converter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// NIRToMarkdown converts NormeInRete (NIR) XML bytes to a well-formatted Markdown string.
func NIRToMarkdown(xmlBytes []byte, vigenza string) (string, error) {
	// HACK: goquery uses an HTML5 parser (net/html) which doesn't support self-closing XML tags.
	xmlStr := string(xmlBytes)
	xmlStr = expandSelfClosingTags(xmlStr)
	xmlStr = expandSelfClosingTags(xmlStr)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(xmlStr))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	footnotes := make(map[string]string) // NIR doesn't seem to use footnotes essentially, but kept for consistency

	if vigenza != "" {
		displayDate := vigenza
		// Try to reformat YYYY-MM-DD to DD-MM-YYYY
		parts := strings.Split(vigenza, "-")
		if len(parts) == 3 {
			displayDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
		sb.WriteString(fmt.Sprintf("*Testo in vigore dal: %s*\n\n", displayDate))
	}

	// 1. Document Title / Intestazione
	// <intestazione><titoloDoc>...</titoloDoc></intestazione>
	docTitle := strings.TrimSpace(doc.Find("intestazione titoloDoc").Text())
	if docTitle != "" {
		sb.WriteString("\n\n<span id=\"preamble\"></span>\n\n")
		sb.WriteString(fmt.Sprintf("# %s\n\n", normalizeWhitespace(docTitle)))
	}

	// 2. Preamble / Formula Iniziale
	// <formulainiziale>...</formulainiziale>
	doc.Find("formulainiziale").Each(func(_ int, s *goquery.Selection) {
		text := processNIRContent(s, &footnotes)
		if strings.TrimSpace(text) != "" {
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	})

	// 3. Body / Articolato
	// <articolato> can contain <capo>, <articolo>, etc.
	doc.Find("articolato").Children().Each(func(i int, s *goquery.Selection) {
		processNIRBodyNode(s, &sb, &footnotes, 1)
	})

	// 4. Attachments / Allegati
	// <annessi><annesso>...</annesso></annessi>
	// or <annessi><annesso id="...">...</annesso>
	// Checking the example file, sometimes they are inside <annessi>
	annessi := doc.Find("annessi annesso")
	if annessi.Length() > 0 {
		sb.WriteString("## Allegati\n\n")
		annessi.Each(func(_ int, annesso *goquery.Selection) {
			processNIRAttachment(annesso, &sb, &footnotes)
		})
	}

	return sb.String(), nil
}

func processNIRBodyNode(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, level int) {
	tagName := goquery.NodeName(s)

	switch tagName {
	case "libro", "parte", "titolo", "capo", "sezione":
		processNIRContainer(s, sb, footnotes, level)
	case "articolo":
		processNIRArticle(s, sb, footnotes, 3)
	default:
		s.Children().Each(func(_ int, child *goquery.Selection) {
			processNIRBodyNode(child, sb, footnotes, level)
		})
	}
}

func processNIRContainer(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, level int) {
	tagName := goquery.NodeName(s)
	num := normalizeWhitespace(s.ChildrenFiltered("num").First().Text())
	rubrica := normalizeWhitespace(s.ChildrenFiltered("rubrica").First().Text())

	fullHeading := num
	if rubrica != "" {
		if fullHeading != "" {
			fullHeading += " - " + rubrica
		} else {
			fullHeading = rubrica
		}
	}

	// Inject Anchor
	if id, exists := s.Attr("id"); exists && id != "" {
		sb.WriteString(fmt.Sprintf(`<span id="%s%s"></span>`, tagName, id) + "\n\n")
	}

	if fullHeading != "" {
		lvl := level
		if lvl > 6 {
			lvl = 6
		}
		prefix := strings.Repeat("#", lvl)
		sb.WriteString(fmt.Sprintf("%s %s\n\n", prefix, fullHeading))
	}

	s.Children().Each(func(_ int, child *goquery.Selection) {
		childTag := goquery.NodeName(child)
		if childTag != "num" && childTag != "rubrica" {
			processNIRBodyNode(child, sb, footnotes, level+1)
		}
	})
}

func processNIRArticle(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, level int) {
	num := strings.TrimSuffix(normalizeWhitespace(s.ChildrenFiltered("num").First().Text()), ".")
	rubrica := normalizeMarkdown(normalizeWhitespace(s.ChildrenFiltered("rubrica").First().Text()))

	// Special handling for Article 1: Preamble might be buried inside
	preambleNodes, cleanArticleNodes, foundInternalHeader := detectAndSplitPreamble(s, num)

	var skipNodes int

	// If we found a split, we render the preamble FIRST
	if len(preambleNodes) > 0 {
		var preambleSb strings.Builder
		for _, node := range preambleNodes {
			// We process these nodes as body content (level 0?)
			// Usually these are <p> tags from the first comma
			t := processNIRContent(node, footnotes)
			if strings.TrimSpace(t) != "" {
				preambleSb.WriteString(t)
				preambleSb.WriteString("\n\n")
			}
		}
		sb.WriteString(preambleSb.String()) // Write Preamble BEFORE Article Header
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
		firstComma := s.ChildrenFiltered("comma").First()
		if firstComma.Length() > 0 {
			rubrica, skipNodes = extractFallbackRubrica(firstComma, num)
		}
	}

	rubrica = normalizeMarkdown(normalizeWhitespace(rubrica))

	// Inject Anchor
	if id, exists := s.Attr("id"); exists && id != "" {
		sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, id) + "\n\n")
	}

	header := fmt.Sprintf("### %s", num)
	if rubrica != "" {
		num = regexp.MustCompile(`^(Art\.\s*\d+)[\s\-]([a-z]+)`).ReplaceAllString(num, "$1-$2")
		rubrica = strings.TrimPrefix(rubrica, num)
		rubrica = normalizeMarkdown(normalizeWhitespace(rubrica))
		header = fmt.Sprintf("### %s - %s", num, rubrica)
	}
	sb.WriteString(header + "\n\n")

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

		var commSb strings.Builder
		for _, node := range cleanArticleNodes {
			t := processNIRContent(node, footnotes)
			if strings.TrimSpace(t) != "" {
				commSb.WriteString(t)
				commSb.WriteString("\n\n")
			}
		}
		if commSb.Len() > 0 {
			sb.WriteString(commSb.String())
		}

		// 2. Process REMAINING commas (after the first one)
		s.Children().Each(func(i int, child *goquery.Selection) {
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

			processNIRArticleChild(child, sb, footnotes, 0, num, rubrica)
		})

	} else {
		// Standard path
		s.Children().Each(func(i int, child *goquery.Selection) {
			tagName := goquery.NodeName(child)
			if tagName == "num" || tagName == "rubrica" {
				return
			}

			// If this is the first comma and we found a fallback rubrica, tell it to skip nodes
			skip := 0
			if i == 0 && skipNodes > 0 && tagName == "comma" {
				skip = skipNodes
			}

			processNIRArticleChild(child, sb, footnotes, skip, num, rubrica)
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
func detectAndSplitPreamble(s *goquery.Selection, articleNum string) ([]*goquery.Selection, []*goquery.Selection, bool) {
	if !strings.Contains(articleNum, "1") {
		return nil, nil, false // Only check Art 1 (or close to it)
	}

	firstComma := s.ChildrenFiltered("comma").First()
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

func processNIRArticleChild(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, skipNodes int, articleNum string, rubrica string) {
	tagName := goquery.NodeName(s)

	if tagName == "comma" {
		processNIRComma(s, sb, footnotes, skipNodes, articleNum, rubrica)
	} else {
		// e.g. direct text or other tags?
		// Usually articles have commas. If strictly <corpo>...
		text := processNIRContent(s, footnotes)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	}
}

func processNIRComma(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string, skipNodes int, articleNum string, rubrica string) {
	//num := normalizeWhitespace(s.ChildrenFiltered("num").First().Text())

	// Inject Anchor
	//if id, exists := s.Attr("id"); exists && id != "" {
	//	sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, id) + "\n\n")
	//}

	contentSel := s.ChildrenFiltered("corpo")
	if contentSel.Length() == 0 {
		// Fallback if no <corpo>, take everything except <num>
		contentSel = s
	}

	// We gather the text from <corpo> or the comma itself
	// but we must skip <num>

	var contentParts []string

	// Helper to extract content excluding num
	extractContent := func(sel *goquery.Selection) {
		sel.Contents().Each(func(i int, node *goquery.Selection) {
			if i < skipNodes {
				return
			}
			if goquery.NodeName(node) == "num" {
				return
			}
			t := processNIRContent(node, footnotes)
			cleanText := strings.ReplaceAll(normalizeMarkdown(normalizeWhitespace(t)), "(", "")
			cleanText = strings.ReplaceAll(cleanText, ")", "")
			cleanText = strings.ReplaceAll(cleanText, "*", "")
			cleanRubrica := strings.ReplaceAll(rubrica, "(", "")
			cleanRubrica = strings.ReplaceAll(cleanRubrica, ")", "")
			cleanRubrica = strings.ReplaceAll(cleanRubrica, "*", "")
			if strings.HasPrefix(cleanText, strings.TrimSuffix(articleNum, ".")) || strings.HasSuffix(cleanText, cleanRubrica) {
				return
			}
			if strings.TrimSpace(t) != "" {
				contentParts = append(contentParts, t)
			}
		})
	}

	if contentSel.Length() > 0 {
		if contentSel.Is("corpo") {
			// Note: extractContent logic assumes 'sel' IS the container whose children we iterate.
			// But contentSel here IS the <corpo> element (a selection).
			// So extractContent iterates its children. Correct.
			extractContent(contentSel)
		} else {
			// Iterate children of comma
			extractContent(s)
		}
	}

	text := strings.Join(contentParts, "")
	text = strings.TrimSpace(text)

	/*
		if num != "" {
			// Escape trailing dot
			cleanNum := strings.TrimSuffix(num, ".")
			sb.WriteString(fmt.Sprintf("**%s\\.** %s\n\n", cleanNum, text))
		} else {
			// TODO: check if text starts with a number followed by optional
			// dash and "bis", "ter", "quater", etc.
			// If so, we should escape the dot after the number.
			sb.WriteString(fmt.Sprintf("%s\n\n", text))
		}
	*/
	sb.WriteString(fmt.Sprintf("%s\n\n", text))
}

func processNIRContent(s *goquery.Selection, footnotes *map[string]string) string {
	// 1. Text Node
	if len(s.Nodes) > 0 && s.Nodes[0].Type == html.TextNode {
		t := strings.TrimSpace(s.Text())
		t = strings.ReplaceAll(t, "<![CDATA[", "> ")
		t = strings.ReplaceAll(t, "]]>", "")
		return t
	}

	// 2. Element Node - Check Tag
	tagName := goquery.NodeName(s)
	cleanTag := strings.TrimPrefix(tagName, "h:")

	switch cleanTag {
	case "p", "div":
		// Indentation handling
		style, _ := s.Attr("h:style")
		if style == "" {
			style, _ = s.Attr("style")
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

		content := processNIRInner(s, footnotes)
		if prefix != "" {
			return prefix + content + "\n\n"
		}
		return content + "\n\n"

	case "br":
		return "\n\n"

	case "table":
		var sb strings.Builder
		processTable(s, &sb)
		return sb.String()

	case "a":
		href, _ := s.Attr("href")
		text := processNIRInner(s, footnotes)
		if href != "" {
			return fmt.Sprintf("[%s](%s)", text, href)
		}
		return text

	case "span", "b", "strong", "i", "em", "testata", "denAnnesso", "titAnnesso":
		// Formatting could be added here, for now pass through
		return processNIRInner(s, footnotes)

	case "ndr":
		text := s.AttrOr("value", "")
		if text == "" {
			text = processNIRInner(s, footnotes)
		}
		return text

	default:
		return processNIRInner(s, footnotes)
	}
}

func processNIRInner(s *goquery.Selection, footnotes *map[string]string) string {
	var sb strings.Builder
	s.Contents().Each(func(_ int, child *goquery.Selection) {
		t := processNIRContent(child, footnotes)
		sb.WriteString(t)
		sb.WriteString("\n\n")
	})
	reNum := regexp.MustCompile(`^([\(]*\d+[\.\d]*[a-z\-]*)\.\s*`)
	text := normalizeMarkdown(normalizeWhitespace(sb.String()))
	text = reNum.ReplaceAllString(text, "**$1\\.** ")

	return text
}

func processNIRAttachment(s *goquery.Selection, sb *strings.Builder, footnotes *map[string]string) {
	// <annesso id="...">
	//   <testata>
	//     <denAnnesso>Allegato A</denAnnesso>
	//     <titAnnesso>...</titAnnesso>
	//   </testata>
	//   <rifesterno link="..."/> OR Content
	// </annesso>

	den := normalizeWhitespace(s.Find("testata denAnnesso").Text())
	tit := normalizeWhitespace(s.Find("testata titAnnesso").Text())

	heading := den
	if tit != "" {
		if heading != "" {
			heading += " - " + tit
		} else {
			heading = tit
		}
	}

	if id, exists := s.Attr("id"); exists && id != "" {
		sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, id) + "\n\n")
	}

	sb.WriteString(fmt.Sprintf("### %s\n\n", heading))

	// Content could be linking to external PDF or embedded
	// In the provided example, we don't see attachments structure deep dive.
	// Assuming generic content processing:
	s.Children().Each(func(_ int, child *goquery.Selection) {
		if goquery.NodeName(child) != "testata" && goquery.NodeName(child) != "meta" {
			// e.g. if it has <rifesterno>
			if goquery.NodeName(child) == "rifesterno" {
				link, _ := child.Attr("xlink:href") // usually URN
				sb.WriteString(fmt.Sprintf("[Vedi Allegato](%s)\n\n", link))
			} else {
				// Process as body node
				processNIRBodyNode(child, sb, footnotes, 4)
			}
		}
	})
}
