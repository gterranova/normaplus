package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gterranova/normattiva-search/internal/normattiva"
)

func main() {
	client := normattiva.NewClient(30 * time.Second)

	fmt.Println("Searching for 'Costituzione'...")
	results, err := client.Search("Costituzione")
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Found %d results\n", len(results))
	if len(results) == 0 {
		log.Fatal("Expected results, found none")
	}

	first := results[0]
	fmt.Printf("First result: %+v\n", first)

	fmt.Println("Fetching XML for first result...")
	xmlBytes, err := client.FetchXML(first.CodiceRedazionale, first.DataPubblicazioneGazzetta, "")
	if err != nil {
		log.Fatalf("FetchXML failed: %v", err)
	}

	fmt.Printf("XML Fetched (%d bytes)\n", len(xmlBytes))
	if len(xmlBytes) < 100 {
		log.Fatal("XML too short")
	}

	// Print first 200 chars
	fmt.Println(string(xmlBytes[:200]))

	// Save for debug
	if err := os.WriteFile("sample.xml", xmlBytes, 0644); err != nil {
		log.Fatal(err)
	}
}
