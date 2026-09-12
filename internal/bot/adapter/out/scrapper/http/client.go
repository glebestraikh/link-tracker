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
	"strconv"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
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

func (c *Client) RegisterChat(ctx context.Context, chatID int64) error {
	reqURL, err := c.joinURL("/tg-chat", strconv.FormatInt(chatID, 10))
	if err != nil {
		return fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(bodyCloser io.ReadCloser) {
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			c.logger.Error("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusConflict {
			return model.ErrChatAlreadyRegistered
		}
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register chat failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("chat registered with scrapper", slog.Int64("chat_id", chatID))
	return nil
}

// DeleteChat - it was in the openapi contract,
// but it is not used in the bot on task (в след лабе скорее всего будет нужно)
func (c *Client) DeleteChat(ctx context.Context, chatID int64) error {
	reqURL, err := c.joinURL("/tg-chat", strconv.FormatInt(chatID, 10))
	if err != nil {
		return fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(bodyCloser io.ReadCloser) {
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			c.logger.Error("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return model.ErrChatNotFound
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete chat failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("chat deleted in scrapper", slog.Int64("chat_id", chatID))
	return nil
}

func (c *Client) AddLink(ctx context.Context, chatID int64, url string, tags []string) (*model.Link, error) {
	reqBody := addLinkRequest{
		Link: url,
		Tags: tags,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL, err := c.joinURL("/links")
	if err != nil {
		return nil, fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(bodyCloser io.ReadCloser) {
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			c.logger.Error("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusConflict {
		return nil, model.ErrLinkAlreadyTracked
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, model.ErrChatNotFound
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("add link failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var linkResp linkResponse
	if err = json.NewDecoder(resp.Body).Decode(&linkResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &model.Link{
		ID:   linkResp.ID,
		URL:  linkResp.URL,
		Tags: linkResp.Tags,
	}, nil
}

func (c *Client) RemoveLink(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	reqBody := removeLinkRequest{Link: url}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL, err := c.joinURL("/links")
	if err != nil {
		return nil, fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(bodyCloser io.ReadCloser) {
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			c.logger.Error("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, model.ErrLinkOrChatNotFound
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("remove link failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var linkResp linkResponse
	if err = json.NewDecoder(resp.Body).Decode(&linkResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &model.Link{
		ID:   linkResp.ID,
		URL:  linkResp.URL,
		Tags: linkResp.Tags,
	}, nil
}

func (c *Client) GetLinks(ctx context.Context, chatID int64) ([]*model.Link, error) {
	reqURL, err := c.joinURL("/links")
	if err != nil {
		return nil, fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(bodyCloser io.ReadCloser) {
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			c.logger.Error("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, model.ErrChatNotFound
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get links failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var listResp listLinksResponse
	if err = json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	links := make([]*model.Link, 0, len(listResp.Links))
	for _, l := range listResp.Links {
		links = append(links, &model.Link{
			ID:   l.ID,
			URL:  l.URL,
			Tags: l.Tags,
		})
	}
	return links, nil
}

func (c *Client) joinURL(pathSegments ...string) (string, error) {
	result, err := url.JoinPath(c.baseURL, pathSegments...)
	if err != nil {
		return "", fmt.Errorf("failed to join url path: %w", err)
	}
	return result, nil
}
