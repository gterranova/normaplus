package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gterranova/normaplus/backend/normattiva"
)

func main() {
	client := normattiva.NewClient(30 * time.Second)

	query := "dlgs 190/2024"
	fmt.Printf("Searching for '%s'...\n", query)
	results, err := client.Search(query)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Found %d results\n", len(results))
	for i, result := range results {
		fmt.Printf("\n[%d] Title: %s\n", i+1, result.Title)
		fmt.Printf("    Code: %s\n", result.CodiceRedazionale)
		fmt.Printf("    Date: %s\n", result.DataPubblicazioneGazzetta)
		fmt.Printf("    Link: %s\n", result.Link)
	}

	// If no results, save the HTML to inspect
	if len(results) == 0 {
		fmt.Println("\nNo results found. Saving HTML response for inspection...")
	}
}
