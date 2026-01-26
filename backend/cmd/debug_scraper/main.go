package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	// 1. Visit Home Page to get cookies
	homeReq, _ := http.NewRequest("GET", "https://www.normattiva.it/", nil)
	homeReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	homeResp, err := client.Do(homeReq)
	if err != nil {
		log.Fatal(err)
	}
	homeResp.Body.Close()
	fmt.Println("Visited home page, cookies:", jar.Cookies(homeReq.URL))

	// 2. Perform Search
	apiURL := "https://www.normattiva.it/ricerca/veloce/0"
	data := url.Values{}
	data.Set("testoRicerca", "Costituzione")

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://www.normattiva.it/")
	req.Header.Set("Origin", "https://www.normattiva.it")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Search Status: %s\n", resp.Status)

	outFile, err := os.Create("search_result.html")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Saved search_result.html")
}
