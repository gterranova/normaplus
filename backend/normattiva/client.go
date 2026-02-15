package normattiva

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gterranova/normaplus/backend/internal/xmlparser"
	"github.com/gterranova/normaplus/backend/normattiva/document"
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

	//fmt.Println("DEBUG Search Status:", resp.Status)

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

func (c *Client) FetchByURN(urn string) (*document.Document, error) {
	code, name, date, err := c.ResolveURN(urn)
	if err != nil {
		return nil, err
	}
	return c.Fetch(code, name, date, "")
}

// Fetch fetches the data for a given document.
func (c *Client) Fetch(codiceRedazionale, name, date, vigenza string) (*document.Document, error) {

	// Default vigenza to today if empty
	if vigenza == "" {
		vigenza = time.Now().Format("2006-01-02")
	}

	// Snap vigenza to publication date if it's earlier
	// Publication date is 'date' (YYYY-MM-DD or YYYYMMDD)
	if pubDate, err := time.Parse("2006-01-02", date); err == nil {
		if vigDate, err := time.Parse("2006-01-02", vigenza); err == nil {
			if vigDate.Before(pubDate) {
				//fmt.Printf("DEBUG: Snapping vigenza %s to publication date %s\n", vigenza, date)
				vigenza = date
			}
		}
	}

	// Cache Logic
	cacheDir := "cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		//fmt.Println("Warning: failed to create cache dir:", err)
	} else {
		result, err := c.retrieveFromCache(codiceRedazionale, vigenza, cacheDir)
		if err == nil && result != nil {
			return result, nil
		}
	}

	if name == "" || date == "" {
		results, err := c.Search(codiceRedazionale)
		if err != nil {
			return nil, err
		}
		if len(results) > 0 {
			name = results[0].Title
			date = results[0].DataPubblicazioneGazzetta
		}
	}

	data, err := c.FetchXML(codiceRedazionale, date, vigenza)
	if err != nil {
		return nil, err
	}

	doc := document.NewDocument(codiceRedazionale, name, date, vigenza)
	if err := xmlparser.FromXML(&doc, data); err != nil {
		return nil, err
	}

	// Save to cache
	c.saveToCache(doc, cacheDir)

	return &doc, nil
}

func (*Client) saveToCache(doc document.Document, cacheDir string) {
	//dateParam := strings.ReplaceAll(doc.DataGU, "-", "")
	vigenzaParam := strings.ReplaceAll(doc.Vigenza, "-", "")
	//filename := fmt.Sprintf("%s_%s_%s.json", dateParam, doc.CodiceRedazionale, vigenzaParam)
	filename := fmt.Sprintf("%s_%s.json", doc.CodiceRedazionale, vigenzaParam)
	cachePath := filepath.Join(cacheDir, filename)
	jsonData, err := doc.ToJSON()
	if err != nil {
		//fmt.Println("Warning: failed to convert to JSON:", err)
	}
	if err := os.WriteFile(cachePath, jsonData, 0644); err != nil {
		//fmt.Println("Warning: failed to write cache:", err)
	}
}

func (*Client) retrieveFromCache(codiceRedazionale, vigenza, cacheDir string) (*document.Document, error) {
	// Normalize dates to YYYYMMDD
	vigenzaParam := strings.ReplaceAll(vigenza, "-", "")
	//filename := fmt.Sprintf("%s_%s_%s.json", dateParam, codiceRedazionale, vigenzaParam)
	filename := fmt.Sprintf("%s_%s.json", codiceRedazionale, vigenzaParam)
	cachePath := filepath.Join(cacheDir, filename)

	info, err := os.Stat(cachePath)
	if err == nil {
		// Check TTL (1 day)
		if time.Since(info.ModTime()) < 24*time.Hour {
			data, err := os.ReadFile(cachePath)
			if err == nil {
				//fmt.Println("Cache hit for", filename)
				doc := document.Document{}
				if err := json.Unmarshal(data, &doc); err != nil {
					return nil, err
				}
				return &doc, nil
			}
		}
	}
	return nil, nil
}

func (c *Client) FetchXML(codiceRedazionale, date, vigenza string) ([]byte, error) {
	if err := c.ensureCookies(); err != nil {
		return nil, fmt.Errorf("failed to init cookies: %w", err)
	}

	// Endpoint: /do/atto/caricaAKN?dataGU=...&codiceRedaz=...&dataVigenza=...

	// 1. Visit Detail Page first (to set session state)
	detailParams := url.Values{}
	detailParams.Set("atto.dataPubblicazioneGazzetta", date) // "1947-12-27"
	detailParams.Set("atto.codiceRedazionale", codiceRedazionale)
	if vigenza == "" {
		vigenza = time.Now().Format("2006-01-02")
	}
	detailParams.Set("atto.dataVigenza", vigenza)
	detailURL := fmt.Sprintf("%s/atto/caricaDettaglioAtto?%s", baseURL, detailParams.Encode())

	//fmt.Printf("DEBUG: Visiting detail page: %s\n", detailURL)
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
	defer detailResp.Body.Close()

	if detailResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("normattiva error: %s", detailResp.Status)
	}

	// Read response
	detailData, err := io.ReadAll(detailResp.Body)
	if err != nil {
		return nil, err
	}

	var data []byte

	if strings.Contains(string(detailData), "/do/atto/caricaAKN") {
		data, err = c.fetchAKNXML(codiceRedazionale, date, vigenza)
	} else {
		data, err = c.fetchPlainXML(codiceRedazionale, date, vigenza)
	}

	if err != nil {
		return nil, err
	}
	return data, nil
}

// fetchAKNXML fetches the Akoma Ntoso XML for a given document.
func (c *Client) fetchAKNXML(codiceRedazionale, date, vigenza string) ([]byte, error) {

	// Normalize dates to YYYYMMDD
	dateParam := strings.ReplaceAll(date, "-", "")
	vigenzaParam := strings.ReplaceAll(vigenza, "-", "")

	detailParams := url.Values{}
	detailParams.Set("atto.dataPubblicazioneGazzetta", date) // "1947-12-27"
	detailParams.Set("atto.codiceRedazionale", codiceRedazionale)
	if vigenza != "" {
		detailParams.Set("atto.dataVigenza", vigenza)
	}
	detailURL := fmt.Sprintf("%s/atto/caricaDettaglioAtto?%s", baseURL, detailParams.Encode())

	// 2. Fetch XML
	params := url.Values{}
	params.Set("dataGU", dateParam)
	params.Set("codiceRedaz", codiceRedazionale)
	params.Set("dataVigenza", vigenzaParam)

	xmlURL := fmt.Sprintf("%s/do/atto/caricaAKN?%s", baseURL, params.Encode())
	//fmt.Printf("DEBUG: Fetching XML: %s\n", xmlURL)

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

	//fmt.Printf("DEBUG: XML Fetch Status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch XML: %s", resp.Status)
	}

	// Read body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Validation: Check if it's actually XML (Normattiva sometimes returns HTML error pages with 200 OK)
	bodyStr := strings.TrimSpace(string(data))
	if !strings.HasPrefix(bodyStr, "<?xml") && strings.HasPrefix(bodyStr, "<!DOCTYPE") {
		data, err = c.fetchPlainXML(codiceRedazionale, date, vigenza)
		if err != nil {
			return nil, fmt.Errorf("normattiva session error: returned HTML instead of XML. Try refreshing the page.")
		}
	}

	if len(data) < 100 {
		return nil, fmt.Errorf("normattiva error: empty or too small response")
	}

	return data, nil
}

// ResolveURN resolves a Normattiva URN to its Codice Redazionale and Date.
// Checks if the response contains a link to the detail page (since Normattiva often returns a list/search result for URNs).
func (c *Client) ResolveURN(urn string) (string, string, string, error) {
	targetURL := fmt.Sprintf("%s/uri-res/N2Ls?%s", baseURL, urn)
	if strings.Contains(urn, "://") {
		targetURL = urn
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	// Parse HTML to find the link
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse URN response: %w", err)
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

	title := doc.Find("title").Text()
	title = strings.TrimSuffix(title, " - Normattiva")
	title = strings.TrimSpace(title)

	if foundHref == "" {
		return "", "", "", fmt.Errorf("could not resolve URN: no matching document found in %s", targetURL)
	}

	// Make it absolute if needed (though we just need params)
	if !strings.HasPrefix(foundHref, "http") {
		foundHref = baseURL + foundHref
	}

	parsedURL, err := url.Parse(foundHref)
	if err != nil {
		return "", "", "", err
	}

	code := parsedURL.Query().Get("atto.codiceRedazionale")
	date := parsedURL.Query().Get("atto.dataPubblicazioneGazzetta")

	if code == "" {
		return "", "", "", fmt.Errorf("could not extract ID from resolved URL: %s", foundHref)
	}

	return code, title, date, nil
}

// fetchPlainXML attempts to fetch XML via the /do/atto/export endpoint
// This is used as a fallback for documents that don't have AKN format available
func (c *Client) fetchPlainXML(codiceRedazionale, date, vigenza string) ([]byte, error) {
	if err := c.ensureCookies(); err != nil {
		return nil, fmt.Errorf("failed to init cookies: %w", err)
	}
	//fmt.Printf("DEBUG: Attempting plain XML export for %s (%s) vigenza=%s\n", codiceRedazionale, date, vigenza)

	// Parse vigenza date for form fields
	vigenzaDate, err := time.Parse("2006-01-02", vigenza)
	if err != nil {
		// Fall back to current date if parsing fails
		vigenzaDate = time.Now()
	}

	// Build POST form data for /do/atto/export
	formData := url.Values{}

	formData.Set("giornoVigenza", fmt.Sprintf("%d", vigenzaDate.Day()))
	formData.Set("meseVigenza", fmt.Sprintf("%d", vigenzaDate.Month()))
	formData.Set("annoVigenza", fmt.Sprintf("%d", vigenzaDate.Year()))
	formData.Set("generaXml", "Esporta XML")
	formData.Set("dataPubblicazioneGazzetta", date)
	formData.Set("codiceRedazionale", codiceRedazionale)
	formData.Set("contenutoForm", "")

	exportURL := fmt.Sprintf("%s/do/atto/export", baseURL)
	//fmt.Printf("DEBUG: POSTing to export endpoint: %s data: %s\n", exportURL, formData.Encode())

	req, err := http.NewRequest("POST", exportURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", fmt.Sprintf("%s/atto/vediMenuExport?atto.dataPubblicazioneGazzetta=%s&atto.codiceRedazionale=%s",
		baseURL, date, codiceRedazionale))
	req.Header.Set("Origin", baseURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//fmt.Printf("DEBUG: Export endpoint Status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to export XML: %s", resp.Status)
	}

	// Read response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Validation: Check if it's actually XML
	bodyStr := strings.TrimSpace(string(data))
	if !strings.HasPrefix(bodyStr, "<?xml") && strings.HasPrefix(bodyStr, "<!DOCTYPE") {
		return nil, fmt.Errorf("export endpoint returned HTML instead of XML (document may not have XML export available)")
	}

	if len(data) < 100 {
		return nil, fmt.Errorf("export endpoint returned empty or too small response")
	}

	//fmt.Printf("DEBUG: Successfully fetched plain XML (%d bytes)\n", len(data))
	return data, nil
}
