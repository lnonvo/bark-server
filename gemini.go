package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mritd/logger"
)

var (
	geminiKey   string
	geminiModel string
)

func SetGeminiConfig(key, model string) {
	geminiKey = key
	geminiModel = model
	if key != "" {
		logger.Infof("Gemini verification code analysis enabled, model: %s", model)
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// analyzeVerificationCode calls the Gemini API to extract a verification code
// from the given text. Returns the code if found, or empty string on any failure.
// Failures are logged and silently ignored so the push flow is never blocked.
func analyzeVerificationCode(body string) string {
	if geminiKey == "" || body == "" {
		return ""
	}

	prompt := fmt.Sprintf(
		"Extract the verification code from the following message. "+
			"Reply with ONLY the verification code itself (digits/letters), nothing else. "+
			"If there is no verification code, reply with exactly \"NONE\".\n\n"+
			"Message: %s", body)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Errorf("Gemini: failed to marshal request: %v", err)
		return ""
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, geminiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		logger.Errorf("Gemini: failed to create request: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("Gemini: request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Errorf("Gemini: API returned status %d: %s", resp.StatusCode, string(respBody))
		return ""
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		logger.Errorf("Gemini: failed to decode response: %v", err)
		return ""
	}

	if len(geminiResp.Candidates) == 0 ||
		len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return ""
	}

	code := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)

	if code == "" || strings.EqualFold(code, "NONE") {
		return ""
	}

	logger.Infof("Gemini: extracted verification code from message")
	return code
}
