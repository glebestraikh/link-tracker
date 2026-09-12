package github

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct {
	handler func(*http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req), nil
}

func TestGithubClient_GetLastUpdated(t *testing.T) {
	t.Parallel()

	t.Run("Success: returns last updated time", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockTransport := &mockRoundTripper{
			handler: func(_ *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"id": 1234567,
						"name": "test-repo",
						"updated_at": "2025-12-01T10:00:00Z",
						"pushed_at": "2025-12-01T09:00:00Z"
					}`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			token:      "",
			hasToken:   false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://github.com/user/repo")

		// Then
		require.NoError(t, err)
		require.False(t, result.IsZero())
	})

	t.Run("Error: non-2xx HTTP status code", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockTransport := &mockRoundTripper{
			handler: func(_ *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			token:      "",
			hasToken:   false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://github.com/user/repo")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "GitHub API error")
	})

	t.Run("Error: invalid JSON response body", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockTransport := &mockRoundTripper{
			handler: func(_ *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{invalid json`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			token:      "",
			hasToken:   false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://github.com/user/repo")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("Success: handles missing optional fields gracefully", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockTransport := &mockRoundTripper{
			handler: func(_ *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"id": 1234567,
						"name": "test-repo"
					}`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			token:      "",
			hasToken:   false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://github.com/user/repo")

		// Then
		require.NoError(t, err)
		require.True(t, result.IsZero(), "missing optional fields should return zero time")
	})

	t.Run("Error: invalid GitHub URL", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		client := &Client{
			httpClient: &http.Client{Timeout: 10 * time.Second},
			token:      "",
			hasToken:   false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://github.com/invalid")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "invalid GitHub URL")
	})
}
