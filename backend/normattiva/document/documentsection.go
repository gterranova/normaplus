package document

import (
	"fmt"
	"regexp"
	"strings"
)

type DocumentSection struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Children []DocumentSection `json:"children,omitempty"`
	Content  []string          `json:"content,omitempty"`
	Root     *Document         `json:"-"`
}

type Attachment struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func NewDocumentSection(sectionType, title string, root *Document) DocumentSection {
	return DocumentSection{
		Type:  sectionType,
		Title: title,
		Root:  root,
	}
}

func (s *DocumentSection) AddContent(content string) {
	s.Content = append(s.Content, content)
}

func (s *DocumentSection) AddSection(doc DocumentSection) {
	s.Children = append(s.Children, doc)
}

func (s *DocumentSection) WriteMarkdown(sb *strings.Builder, level int) {
	// 1. Title/Header
	if s.Title != "" {
		if s.ID != "" {
			sb.WriteString(fmt.Sprintf(`<span id="%s"></span>`, s.ID) + "\n\n")
		}

		// If level > 6, cap at 6
		lvl := level
		if lvl > 6 {
			lvl = 6
		}
		prefix := strings.Repeat("#", lvl)

		sb.WriteString(fmt.Sprintf("%s %s\n\n", prefix, s.Title))
	} else if s.Type == "preamble" {
		// Preamble might not have a title but acts as a block
	}

	// 2. Content
	newCommaRe := regexp.MustCompile(`^[\(\s]*\d+[a-z-]*\\?\.[\s\)\)]+`)
	possibleInsRe := regexp.MustCompile(`\(\(([^)]+|[^)]*\)\s[^)]*)\)\)`)
	for _, content := range s.Content {
		possibleNewComma := newCommaRe.FindString(content)
		if len(possibleNewComma) > 0 {
			// remove possibleNewComma from content
			content = content[len(possibleNewComma):]
			// add possibleNewComma to sb
			sb.WriteString("**" + strings.TrimSpace(possibleNewComma) + "** ")
		}

		content = possibleInsRe.ReplaceAllString(content, "**(($1))**")

		sb.WriteString(content + "\n\n")
	}

	// 3. Children
	// Calculate next level
	nextLevel := level + 1
	// Some container types might stay at same level?
	// Usually hierarchy implies deeper level.
	// Exception: "body" -> "libri" -> "titles".
	// If current section is just a wrapper (like "body"), maybe don't increase?
	// But our DocumentSection usually represents a structural node.
	if s.Type == "body" {
		nextLevel = level // Body doesn't add a header itself usually
	}

	for _, child := range s.Children {
		child.WriteMarkdown(sb, nextLevel)
	}
}
