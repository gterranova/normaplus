package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gterranova/normaplus/backend/internal/xmlparser"
	"github.com/gterranova/normaplus/backend/normattiva"
	"github.com/gterranova/normaplus/backend/normattiva/document"
)

func main() {
	client := normattiva.NewClient(0)

	// Test document that should NOT have AKN format
	codice := "23G00195"
	date := "2023-12-09"
	vigenza := "2024-01-30"

	fmt.Printf("Testing document: %s (%s)\\n", codice, date)

	xmlData, err := client.FetchXML(codice, date, vigenza)
	if err != nil {
		log.Fatalf("Error fetching XML: %v", err)
	}

	// Check if it's HTML error or actual XML
	if len(xmlData) > 100 {
		preview := string(xmlData[:100])
		fmt.Printf("First 100 bytes: %s\\n", preview)
	}

	// Save to file for inspection
	err = os.WriteFile("test_23G00195.xml", xmlData, 0644)
	if err != nil {
		log.Fatalf("Error writing XML file: %v", err)
	}
	fmt.Println("XML saved to test_23G00195.xml")

	// Test Conversion
	doc := document.NewDocument(codice, "", date, vigenza)
	err = xmlparser.FromXML(&doc, xmlData)
	if err != nil {
		log.Fatalf("Error converting to Markdown: %v", err)
	}

	md, err := doc.ToMarkdown()
	if err != nil {
		log.Fatalf("Error converting to Markdown: %v", err)
	}

	err = os.WriteFile("test_23G00195.md", []byte(md), 0644)
	if err != nil {
		log.Fatalf("Error writing MD file: %v", err)
	}
	fmt.Println("Markdown saved to test_23G00195.md")
}
