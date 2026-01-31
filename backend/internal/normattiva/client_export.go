package normattiva

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchPlainXML attempts to fetch XML via the /do/atto/export endpoint
// This is used as a fallback for documents that don't have AKN format available
func (c *Client) FetchPlainXML(codiceRedazionale, date, vigenza string) ([]byte, error) {
	fmt.Printf("DEBUG: Attempting plain XML export for %s (%s) vigenza=%s\n", codiceRedazionale, date, vigenza)

	// 1. Visit Detail Page first (to set session state)
	/*
		detailParams := url.Values{}
		detailParams.Set("action", "select-all")
		detailParams.Set("atto.dataPubblicazioneGazzetta", date)
		detailParams.Set("atto.codiceRedazionale", codiceRedazionale)
		if vigenza != "" {
			detailParams.Set("atto.dataVigenza", vigenza)
		}
		detailURL := fmt.Sprintf("%s/atto/caricaDettaglioAtto?%s", baseURL, detailParams.Encode())
		menuExportURL := fmt.Sprintf("%s/atto/vediMenuExport?%s", baseURL, detailParams.Encode())

		fmt.Printf("DEBUG: Visiting menu export page: %s\n", menuExportURL)
		menuExportReq, err := http.NewRequest("GET", menuExportURL, nil)
		if err != nil {
			return nil, err
		}
		menuExportReq.Header.Set("User-Agent", userAgent)
		menuExportReq.Header.Set("Referer", detailURL)
		menuExportResp, err := c.httpClient.Do(menuExportReq)
		if err != nil {
			return nil, err
		}
		defer menuExportResp.Body.Close()
		fmt.Printf("DEBUG: Menu export page Status: %s\n", menuExportResp.Status)

		// Read response
		menuExportData, err := io.ReadAll(menuExportResp.Body)
		if err != nil {
			return nil, err
		}

		fmt.Println("DEBUG: Menu export page response:", string(menuExportData))
	*/
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
	fmt.Printf("DEBUG: POSTing to export endpoint: %s data: %s\n", exportURL, formData.Encode())

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

	fmt.Printf("DEBUG: Export endpoint Status: %s\n", resp.Status)

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

	fmt.Printf("DEBUG: Successfully fetched plain XML (%d bytes)\n", len(data))
	return data, nil
}
