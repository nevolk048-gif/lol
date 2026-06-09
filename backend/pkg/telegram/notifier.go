package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Notifier struct {
	token  string
	chatID string
	client *http.Client
}

func NewNotifier(token, chatID string) *Notifier {
	return &Notifier{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled reports whether the notifier is configured.
func (n *Notifier) Enabled() bool {
	return n != nil && n.token != "" && n.chatID != ""
}

// Send sends a plain HTML text message to the configured chat.
func (n *Notifier) Send(ctx context.Context, text string) error {
	if !n.Enabled() {
		return nil
	}

	payload, _ := json.Marshal(map[string]string{
		"chat_id":    n.chatID,
		"text":       text,
		"parse_mode": "HTML",
	})

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}
