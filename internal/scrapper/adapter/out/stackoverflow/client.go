package stackoverflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	baseURL                         = "https://api.stackexchange.com"
	defaultHTTPTimeout              = 10 * time.Second
	minStackOverflowMatchGroupCount = 2
)

var stackOverflowQuestionRegex = regexp.MustCompile(`stackoverflow\.com/questions/(\d+)`)

type Client struct {
	httpClient *http.Client
	apiKey     string
	hasKey     bool
	logger     *slog.Logger
}

func NewClient(apiKey string, hasKey bool, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		apiKey:     apiKey,
		hasKey:     hasKey,
		logger:     logger,
	}
}

func (c *Client) Supports(linkURL string) bool {
	return stackOverflowQuestionRegex.MatchString(linkURL)
}

func (c *Client) GetLastUpdated(ctx context.Context, linkURL string) (time.Time, error) {
	questionID, err := parseQuestionID(linkURL)
	if err != nil {
		return time.Time{}, err
	}

	requestURL, err := c.buildRequestURL(questionID)
	if err != nil {
		return time.Time{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create request: %w", err)
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
		return time.Time{}, fmt.Errorf("StackOverflow API error: status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&apiResp); decodeErr != nil {
		return time.Time{}, fmt.Errorf("failed to decode response: %w", decodeErr)
	}

	if len(apiResp.Items) == 0 {
		return time.Time{}, fmt.Errorf("question not found: %s", questionID)
	}

	lastUpdated := time.Unix(apiResp.Items[0].LastActivityDate, 0)
	c.logger.Info("got StackOverflow last updated", slog.String("url", linkURL), slog.Time("last_updated", lastUpdated))
	return lastUpdated, nil
}

func (c *Client) buildRequestURL(questionID string) (string, error) {
	apiURL, err := url.JoinPath(baseURL, "/2.3/questions", questionID)
	if err != nil {
		return "", fmt.Errorf("failed to build request url: %w", err)
	}

	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse request url: %w", err)
	}

	query := parsedURL.Query()
	query.Set("site", "stackoverflow")
	if c.hasKey && c.apiKey != "" {
		query.Set("key", c.apiKey)
	}
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func parseQuestionID(url string) (string, error) {
	matches := stackOverflowQuestionRegex.FindStringSubmatch(url)
	if len(matches) < minStackOverflowMatchGroupCount {
		return "", fmt.Errorf("invalid StackOverflow URL: %s", url)
	}
	return matches[1], nil
}
