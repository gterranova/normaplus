package xmlparser

import (
	"fmt"
	"os"
	"testing"

	"github.com/gterranova/normaplus/backend/normattiva/document"
)

func TestDocumentToMarkdown(t *testing.T) {
	// Locate sample.xml in backend root
	path := "../../sample.xml"
	xmlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample.xml not found at %s, skipping integration test", path)
	}

	// 1. Parse into Document
	doc := document.NewDocument("", "", "", "")
	err = FromXML(&doc, xmlBytes)
	if err != nil {
		t.Fatalf("FromXML failed: %v", err)
	}

	// 2. Convert to Markdown
	mdBytes, err := doc.ToMarkdown()
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	if len(mdBytes) == 0 {
		t.Fatal("Markdown output is empty")
	}

	fmt.Println("--- converted document markdown sample ---")
	mdStr := string(mdBytes)
	if len(mdStr) > 500 {
		fmt.Println(mdStr[:500])
	} else {
		fmt.Println(mdStr)
	}
	fmt.Println("...")

	os.WriteFile("../../sample_doc.md", mdBytes, 0644)
}
