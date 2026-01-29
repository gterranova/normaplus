package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Provider interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// GeminiProvider implements AI using Google's Gemini API
type GeminiProvider struct {
	APIKey string
	Model  string
}

func (g *GeminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if g.APIKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	payload := map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"parts": []interface{}{
					map[string]interface{}{
						"text": prompt,
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (%d): %s", resp.StatusCode, string(b))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("no response from Gemini")
}

// OllamaProvider implements AI using a local Ollama instance
type OllamaProvider struct {
	Host  string
	Model string
}

func (o *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", o.Host)

	payload := map[string]interface{}{
		"model":  o.Model,
		"prompt": prompt,
		"stream": false,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (%d): %s", resp.StatusCode, string(b))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

// Service manages AI operations
type Service struct {
	provider Provider
}

func NewService() *Service {
	// Try Gemini first
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		fmt.Println("AI: Using Gemini provider")
		return &Service{
			provider: &GeminiProvider{
				APIKey: apiKey,
				Model:  "gemini-1.5-flash",
			},
		}
	}

	// Fallback to Ollama if configured
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}
	fmt.Println("AI: Using Ollama provider at", ollamaHost)
	return &Service{
		provider: &OllamaProvider{
			Host:  ollamaHost,
			Model: "llama3.2:1b", // default model
		},
	}
}

func (s *Service) Summarize(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf("Summarize the following Italian legal text briefly and clearly in Italian:\n\n%s", text)
	return s.provider.Generate(ctx, prompt)
}

func (s *Service) Translate(ctx context.Context, text, targetLang string) (string, error) {
	prompt := fmt.Sprintf("Translate the following Italian legal text to %s. Maintain the legal terminology accuracy:\n\n%s", targetLang, text)
	return s.provider.Generate(ctx, prompt)
}
