package stackoverflow

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

func TestStackOverflowClient_GetLastUpdated(t *testing.T) {
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
						"items": [
							{
								"question_id": 12345,
								"title": "test question",
								"last_activity_date": 1735689600
							}
						]
					}`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			apiKey:     "",
			hasKey:     false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://stackoverflow.com/questions/12345/test")

		// Then
		require.NoError(t, err)
		require.False(t, result.IsZero())
	})

	t.Run("Error: non-2xx HTTP status code (401 Unauthorized)", func(t *testing.T) {
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
			apiKey:     "",
			hasKey:     false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://stackoverflow.com/questions/12345/test")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "StackOverflow API error")
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
					Body:       io.NopCloser(strings.NewReader(`{invalid json}`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			apiKey:     "",
			hasKey:     false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://stackoverflow.com/questions/12345/test")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("Error: empty items array in response", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockTransport := &mockRoundTripper{
			handler: func(_ *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"items": []}`)),
				}
			},
		}

		client := &Client{
			httpClient: &http.Client{Transport: mockTransport, Timeout: 10 * time.Second},
			apiKey:     "",
			hasKey:     false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://stackoverflow.com/questions/12345/test")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "question not found")
	})

	t.Run("Error: invalid StackOverflow URL", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		client := &Client{
			httpClient: &http.Client{Timeout: 10 * time.Second},
			apiKey:     "",
			hasKey:     false,
			logger:     logger,
		}

		// When
		result, err := client.GetLastUpdated(t.Context(), "https://stackoverflow.com/invalid")

		// Then
		require.Error(t, err)
		require.True(t, result.IsZero())
		require.Contains(t, err.Error(), "invalid StackOverflow URL")
	})
}
