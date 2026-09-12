package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL            = "https://api.github.com"
	defaultHTTPTimeout = 10 * time.Second
	minGithubURLParts  = 5
)

type Client struct {
	httpClient *http.Client
	token      string
	hasToken   bool
	logger     *slog.Logger
}

func NewClient(token string, hasToken bool, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		token:      token,
		hasToken:   hasToken,
		logger:     logger,
	}
}

func (c *Client) Supports(url string) bool {
	return strings.Contains(url, "github.com")
}

func (c *Client) GetLastUpdated(ctx context.Context, linkURL string) (time.Time, error) {
	owner, repo, err := parseGithubURL(linkURL)
	if err != nil {
		return time.Time{}, err
	}

	apiURL, err := url.JoinPath(baseURL, "/repos", owner, repo)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	if c.hasToken && c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return time.Time{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			c.logger.Warn("failed to close response body", slog.Any("error", closeErr))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	var repoResp repoResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&repoResp); decodeErr != nil {
		return time.Time{}, fmt.Errorf("failed to decode response: %w", decodeErr)
	}

	lastUpdated := time.Time{}
	if repoResp.UpdatedAt != nil {
		lastUpdated = *repoResp.UpdatedAt
	}

	if repoResp.PushedAt != nil && repoResp.PushedAt.After(lastUpdated) {
		lastUpdated = *repoResp.PushedAt
	}

	c.logger.Info("got GitHub last updated", slog.String("url", linkURL), slog.Time("last_updated", lastUpdated))

	return lastUpdated, nil
}

func parseGithubURL(url string) (owner, repo string, err error) {
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	if len(parts) < minGithubURLParts {
		return "", "", fmt.Errorf("invalid GitHub URL: %s", url)
	}
	return parts[3], parts[4], nil
}
