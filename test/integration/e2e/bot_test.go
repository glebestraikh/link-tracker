//go:build integration

package e2e_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBotUpdateValidRequest(t *testing.T) {
	t.Log("Test 1: Bot accepts valid update request format")

	const chatID = int64(9001)

	payload := map[string]interface{}{
		"id":          int64(1),
		"url":         "https://github.com/user/repo",
		"description": "Repository updated",
		"tgChatIds":   []int64{chatID},
	}

	resp := doJSON(t, "POST", botBaseURL+"/updates", payload, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200 OK for valid update request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 1 PASSED: Bot accepted valid request with status: %d, response: %s", resp.StatusCode, string(body))
}

func TestBotUpdateInvalidRequestMissingFields(t *testing.T) {
	t.Log("Test 2a: Bot rejects request with missing required fields")

	payload := map[string]interface{}{
		"id": int64(1),
		// Missing: url, description, tgChatIds
	}

	resp := doJSON(t, "POST", botBaseURL+"/updates", payload, nil)
	defer resp.Body.Close()

	require.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected non-200 status for invalid request")
	require.True(t, resp.StatusCode >= 400, "Expected 4xx status code for bad request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 2a PASSED: Bot rejected invalid request with status: %d, response: %s", resp.StatusCode, string(body))
}

func TestBotUpdateInvalidRequestWrongType(t *testing.T) {
	t.Log("Test 2b: Bot rejects request with wrong field types")

	payload := map[string]interface{}{
		"id":          "invalid_id_string", // Should be int64
		"url":         "https://example.com",
		"description": 12345, // Should be string
		"tgChatIds":   []int64{123},
	}

	resp := doJSON(t, "POST", botBaseURL+"/updates", payload, nil)
	defer resp.Body.Close()

	require.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected non-200 status for invalid field types")
	require.True(t, resp.StatusCode >= 400, "Expected 4xx status code for bad request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 2b PASSED: Bot rejected invalid types with status: %d, response: %s", resp.StatusCode, string(body))
}

func TestBotUpdateEmptyTgChatIds(t *testing.T) {
	t.Log("Test 2c: Bot rejects request with empty tgChatIds array")

	payload := map[string]interface{}{
		"id":          int64(1),
		"url":         "https://example.com",
		"description": "test",
		"tgChatIds":   []int64{},
	}

	resp := doJSON(t, "POST", botBaseURL+"/updates", payload, nil)
	defer resp.Body.Close()

	require.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected non-200 status for empty tgChatIds")
	require.True(t, resp.StatusCode >= 400, "Expected 4xx status code for bad request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 2c PASSED: Bot rejected empty tgChatIds with status: %d, response: %s", resp.StatusCode, string(body))
}

func TestBotUpdateEmptyURL(t *testing.T) {
	t.Log("Test 2d: Bot rejects request with empty URL")

	payload := map[string]interface{}{
		"id":          int64(1),
		"url":         "",
		"description": "test",
		"tgChatIds":   []int64{123},
	}

	resp := doJSON(t, "POST", botBaseURL+"/updates", payload, nil)
	defer resp.Body.Close()

	require.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected non-200 status for empty URL")
	require.True(t, resp.StatusCode >= 400, "Expected 4xx status code for bad request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 2d PASSED: Bot rejected empty URL with status: %d, response: %s", resp.StatusCode, string(body))
}

func TestBotUpdateEmptyBody(t *testing.T) {
	t.Log("Test 2e: Bot rejects request with empty body")

	resp := doJSON(t, "POST", botBaseURL+"/updates", nil, nil)
	defer resp.Body.Close()

	require.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected non-200 status for empty body")
	require.True(t, resp.StatusCode >= 400, "Expected 4xx status code for bad request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 2e PASSED: Bot rejected empty body with status: %d, response: %s", resp.StatusCode, string(body))
}

func TestBotUpdateUnregisteredChats(t *testing.T) {
	t.Log("Test 2f: Bot rejects request with unregistered chat IDs")

	payload := map[string]interface{}{
		"id":          int64(2),
		"url":         "https://github.com/user/repo",
		"description": "test update",
		"tgChatIds":   []int64{99999, 99998}, // Unregistered chats
	}

	resp := doJSON(t, "POST", botBaseURL+"/updates", payload, nil)
	defer resp.Body.Close()

	require.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected non-200 status for unregistered chats")
	require.True(t, resp.StatusCode >= 400, "Expected 4xx status code for bad request")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Test 2f PASSED: Bot rejected unregistered chats with status: %d, response: %s", resp.StatusCode, string(body))
}
