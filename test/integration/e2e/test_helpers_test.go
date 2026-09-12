//go:build integration

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type listLinksResponse struct {
	Links []linkResponse `json:"links"`
	Size  int32          `json:"size"`
}

type linkResponse struct {
	ID   int64    `json:"id"`
	URL  string   `json:"url"`
	Tags []string `json:"tags"`
}

func doJSON(t *testing.T, method, url string, payload any, headers map[string]string) *http.Response {
	client := &http.Client{Timeout: 10 * time.Second}
	return doRequest(t, client, method, url, payload, headers)
}

func doRequest(t *testing.T, client *http.Client, method, url string, payload any, headers map[string]string) *http.Response {
	t.Helper()

	var body *bytes.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(data)
	} else {
		body = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func projectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot get caller info")
	}

	dir := filepath.Dir(filename)

	return filepath.Abs(filepath.Join(dir, "../../.."))
}
