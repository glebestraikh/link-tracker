package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const defaultHTTPTimeout = 10 * time.Second

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		logger:     logger,
	}
}

func (c *Client) SendUpdate(ctx context.Context, id int64, linkURL string, description string, tgChatIDs []int64) error {
	update := linkUpdate{
		ID:          id,
		URL:         linkURL,
		Description: description,
		TgChatIDs:   tgChatIDs,
	}

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal updates: %w", err)
	}

	reqURL, err := url.JoinPath(c.baseURL, "/updates")
	if err != nil {
		return fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(body io.ReadCloser) {
		closeErr := body.Close()
		if closeErr != nil {
			c.logger.Warn("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send updates failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("updates sent to bot", slog.Int64("link_id", id), slog.String("url", linkURL))

	return nil
}
