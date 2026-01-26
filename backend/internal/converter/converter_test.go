package converter

import (
	"fmt"
	"os"
	"testing"
)

func TestToMarkdown(t *testing.T) {
	// Locate sample.xml in backend root
	path := "../../sample.xml"
	xmlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample.xml not found at %s, skipping integration test", path)
	}

	md, err := ToMarkdown(xmlBytes)
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	if len(md) == 0 {
		t.Fatal("Markdown is empty")
	}

	fmt.Println("--- converted markdown sample ---")
	fmt.Println(md[:500]) // Print first 500 chars
	fmt.Println("...")

	// Save output to verify manually
	os.WriteFile("../../sample.md", []byte(md), 0644)
}
