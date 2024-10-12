package catbox

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	maxRetries = 3
	retryDelay = 2 * time.Second
)

func SendMessageToWebhook(message string) error {
	if G_config.WebhookURL == "" {
		G_logger.Warn("Webhook URL is not configured; skipping webhook notification.")
		return nil
	}

	payload := fmt.Sprintf(`{"content": "%s"}`, message)

	var err error
	for i := 0; i < maxRetries; i++ {
		resp, err := httpClient.Post(G_config.WebhookURL, "application/json", bytes.NewBufferString(payload))
		if err != nil {
			G_logger.Warnf("Failed to send webhook: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				G_logger.Infof("Webhook message sent: %s", message)
				return nil
			} else {
				err = fmt.Errorf("webhook returned status: %s", resp.Status)
				G_logger.Warnf("Attempt %d: %s", i+1, err)
			}
		}

		time.Sleep(retryDelay)
	}

	return err
}
