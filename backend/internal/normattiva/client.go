package normattiva

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
	}
}

type DocumentMetadata struct {
	Title                     string `json:"title"`
	DataPubblicazioneGazzetta string `json:"data_pubblicazione_gazzetta"`
	CodiceRedazionale         string `json:"codice_redazionale"`
	Link                      string `json:"link,omitempty"`
}

const (
	baseURL   = "https://www.normattiva.it"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// ensureCookies visits the home page to establish a session if needed.
func (c *Client) ensureCookies() error {
	req, err := http.NewRequest("GET", baseURL+"/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Search performs a search on Normattiva based on the query string.
func (c *Client) Search(query string) ([]DocumentMetadata, error) {
	if err := c.ensureCookies(); err != nil {
		return nil, fmt.Errorf("failed to init cookies: %w", err)
	}

	data := url.Values{}
	data.Set("testoRicerca", query)

	req, err := http.NewRequest("POST", baseURL+"/ricerca/veloce/0", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("Origin", baseURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("DEBUG Search Status:", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("normattiva returned status %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []DocumentMetadata
	doc.Find("#elenco_risultati .boxAtto").Each(func(i int, s *goquery.Selection) {
		linkSel := s.Find(".collapse-header a")
		title := strings.TrimSpace(linkSel.Text())
		// Normalize whitespace in title
		title = strings.Join(strings.Fields(title), " ")

		href, exists := linkSel.Attr("href")
		if exists {
			href = strings.TrimSpace(href)
			u, err := url.Parse(href)
			if err == nil {
				q := u.Query()
				date := q.Get("atto.dataPubblicazioneGazzetta")
				code := q.Get("atto.codiceRedazionale")

				if date != "" && code != "" {
					results = append(results, DocumentMetadata{
						Title:                     title,
						DataPubblicazioneGazzetta: date,
						CodiceRedazionale:         code,
						Link:                      href,
					})
				}
			}
		}
	})

	return results, nil
}

// FetchXML fetches the Akoma Ntoso XML for a given document.
func (c *Client) FetchXML(codiceRedazionale, date string) ([]byte, error) {
	if err := c.ensureCookies(); err != nil {
		return nil, fmt.Errorf("failed to init cookies: %w", err)
	}

	// date comes as YYYY-MM-DD from Search, but FetchXML needs YYYYMMDD
	dateParam := strings.ReplaceAll(date, "-", "")

	// Endpoint: /do/atto/caricaAKN?dataGU=...&codiceRedaz=...&dataVigenza=...
	// Let's use today's date for vigency to get the current text.
	vigenza := time.Now().Format("20060102")

	// 1. Visit Detail Page first (to set session state)
	// The detail URL structure from search result:
	// /atto/caricaDettaglioAtto?atto.dataPubblicazioneGazzetta=YYYY-MM-DD&atto.codiceRedazionale=...
	detailParams := url.Values{}
	detailParams.Set("atto.dataPubblicazioneGazzetta", date) // "1947-12-27"
	detailParams.Set("atto.codiceRedazionale", codiceRedazionale)
	detailURL := fmt.Sprintf("%s/atto/caricaDettaglioAtto?%s", baseURL, detailParams.Encode())

	detailReq, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil, err
	}
	detailReq.Header.Set("User-Agent", userAgent)
	detailReq.Header.Set("Referer", baseURL+"/ricerca/veloce/0") // Referer from search
	detailResp, err := c.httpClient.Do(detailReq)
	if err != nil {
		return nil, err
	}
	detailResp.Body.Close()

	// 2. Fetch XML
	params := url.Values{}
	params.Set("dataGU", dateParam)
	params.Set("codiceRedaz", codiceRedazionale)
	params.Set("dataVigenza", vigenza)

	xmlURL := fmt.Sprintf("%s/do/atto/caricaAKN?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", xmlURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", detailURL) // Referer from detail page
	req.Header.Set("Origin", baseURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch XML: %s", resp.Status)
	}

	// Read body
	return io.ReadAll(resp.Body)
}

// ResolveURN resolves a Normattiva URN to its Codice Redazionale and Date.
// Checks if the response contains a link to the detail page (since Normattiva often returns a list/search result for URNs).
func (c *Client) ResolveURN(urn string) (string, string, error) {
	targetURL := fmt.Sprintf("%s/uri-res/N2Ls?%s", baseURL, urn)
	if strings.Contains(urn, "://") {
		targetURL = urn
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Parse HTML to find the link
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URN response: %w", err)
	}

	// Look for the first result link
	// Typicall format: <a href="/atto/caricaDettaglioAtto?..." ...>
	var foundHref string
	doc.Find(`a[href*="caricaDettaglioAtto"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, exists := s.Attr("href")
		if exists {
			foundHref = href
			return false // break
		}
		return true
	})

	if foundHref == "" {
		return "", "", fmt.Errorf("could not resolve URN: no matching document found in %s", targetURL)
	}

	// Make it absolute if needed (though we just need params)
	if !strings.HasPrefix(foundHref, "http") {
		foundHref = baseURL + foundHref
	}

	parsedURL, err := url.Parse(foundHref)
	if err != nil {
		return "", "", err
	}

	code := parsedURL.Query().Get("atto.codiceRedazionale")
	date := parsedURL.Query().Get("atto.dataPubblicazioneGazzetta")

	if code == "" {
		return "", "", fmt.Errorf("could not extract ID from resolved URL: %s", foundHref)
	}

	return code, date, nil
}
