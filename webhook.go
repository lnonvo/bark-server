package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/finb/bark-server/v2/apns"
	"github.com/mritd/logger"
)

var webhookURL string

var webhookClient = &http.Client{
	Timeout: 10 * time.Second,
}

func SetWebhookURL(url string) {
	webhookURL = url
	if url != "" {
		logger.Infof("Webhook enabled, URL: %s", url)
	}
}

type webhookPayload struct {
	DeviceKey string                 `json:"device_key"`
	Title     string                 `json:"title,omitempty"`
	Subtitle  string                 `json:"subtitle,omitempty"`
	Body      string                 `json:"body,omitempty"`
	Sound     string                 `json:"sound,omitempty"`
	ExtParams map[string]interface{} `json:"ext_params,omitempty"`
}

// fireWebhook sends the push message details to the configured webhook URL
// asynchronously. It never blocks the caller or affects the push flow.
func fireWebhook(msg *apns.PushMessage) {
	if webhookURL == "" {
		return
	}

	go func() {
		payload := webhookPayload{
			DeviceKey: msg.DeviceKey,
			Title:     msg.Title,
			Subtitle:  msg.Subtitle,
			Body:      msg.Body,
			Sound:     msg.Sound,
			ExtParams: msg.ExtParams,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			logger.Errorf("Webhook: failed to marshal payload: %v", err)
			return
		}

		resp, err := webhookClient.Post(webhookURL, "application/json", bytes.NewReader(jsonData))
		if err != nil {
			logger.Errorf("Webhook: request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			logger.Errorf("Webhook: server returned status %d", resp.StatusCode)
		}
	}()
}
