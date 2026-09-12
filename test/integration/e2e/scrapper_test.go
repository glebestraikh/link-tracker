//go:build integration

package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScrapperLinksFlow(t *testing.T) {
	t.Log("Test 3: Scrapper link and chat management")
	client := &http.Client{}

	t.Run("add and list", func(t *testing.T) {
		chatID := "101"
		resp := doRequest(t, client, "POST", scrapperBaseURL+"/tg-chat/"+chatID, nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Chat registration should return 200")
		resp.Body.Close()

		addPayload := map[string]interface{}{"link": "https://example.com/a", "tags": []string{"tag1"}}
		headers := map[string]string{"Tg-Chat-Id": chatID}
		resp = doRequest(t, client, "POST", scrapperBaseURL+"/links", addPayload, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Adding link should return 200")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NotEmpty(t, body, "Response body should not be empty when adding link")

		resp = doRequest(t, client, "GET", scrapperBaseURL+"/links", nil, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Getting links should return 200")
		var list listLinksResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&list), "Response should be valid JSON")
		resp.Body.Close()

		require.Len(t, list.Links, 1, "Should have exactly 1 link")
		require.Equal(t, int32(1), list.Size, "Size field should be 1")

		link := list.Links[0]
		require.Equal(t, "https://example.com/a", link.URL, "URL should match")
		require.ElementsMatch(t, []string{"tag1"}, link.Tags, "Tags should match")
		require.Greater(t, link.ID, int64(0), "Link ID should be positive")
		require.NotNil(t, link.Tags, "Tags should not be nil")
		require.Len(t, link.Tags, 1, "Should have exactly 1 tag")
	})

	t.Run("add and delete", func(t *testing.T) {
		chatID := "102"
		resp := doRequest(t, client, "POST", scrapperBaseURL+"/tg-chat/"+chatID, nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Chat registration should return 200")
		resp.Body.Close()

		addPayload := map[string]interface{}{"link": "https://example.com/b"}
		headers := map[string]string{"Tg-Chat-Id": chatID}
		resp = doRequest(t, client, "POST", scrapperBaseURL+"/links", addPayload, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Adding link should return 200")
		resp.Body.Close()

		resp = doRequest(t, client, "GET", scrapperBaseURL+"/links", nil, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var listBefore listLinksResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&listBefore))
		resp.Body.Close()
		require.Len(t, listBefore.Links, 1, "Should have 1 link before deletion")

		deletePayload := map[string]interface{}{"link": "https://example.com/b"}
		resp = doRequest(t, client, "DELETE", scrapperBaseURL+"/links", deletePayload, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Deleting link should return 200")
		resp.Body.Close()

		resp = doRequest(t, client, "GET", scrapperBaseURL+"/links", nil, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Getting links after delete should return 200")
		var list listLinksResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
		resp.Body.Close()

		require.Empty(t, list.Links, "Links list should be empty after deletion")
		require.Equal(t, int32(0), list.Size, "Size should be 0 after deletion")
	})

	t.Run("delete from missing chat", func(t *testing.T) {
		chatID := "103"
		resp := doRequest(t, client, "POST", scrapperBaseURL+"/tg-chat/"+chatID, nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		addPayload := map[string]interface{}{"link": "https://example.com/c"}
		headers := map[string]string{"Tg-Chat-Id": chatID}
		resp = doRequest(t, client, "POST", scrapperBaseURL+"/links", addPayload, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Adding link should succeed")
		resp.Body.Close()

		resp = doRequest(t, client, "GET", scrapperBaseURL+"/links", nil, headers)
		var listBefore listLinksResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&listBefore))
		resp.Body.Close()
		require.Len(t, listBefore.Links, 1, "Link should be added")

		missingHeaders := map[string]string{"Tg-Chat-Id": "999"}
		deletePayload := map[string]interface{}{"link": "https://example.com/c"}
		resp = doRequest(t, client, "DELETE", scrapperBaseURL+"/links", deletePayload, missingHeaders)
		require.NotEqual(t, http.StatusOK, resp.StatusCode, "Deleting from missing chat should fail")
		require.True(t, resp.StatusCode >= 400, "Should return 4xx error")
		resp.Body.Close()

		resp = doRequest(t, client, "GET", scrapperBaseURL+"/links", nil, headers)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var list listLinksResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
		resp.Body.Close()

		require.Len(t, list.Links, 1, "Original link should still exist")
		require.Equal(t, "https://example.com/c", list.Links[0].URL, "Link URL should be preserved")
	})

	t.Run("add link for missing chat", func(t *testing.T) {
		chatID := "104"
		resp := doRequest(t, client, "POST", scrapperBaseURL+"/tg-chat/"+chatID, nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		addPayload := map[string]interface{}{"link": "https://example.com/d"}
		missingHeaders := map[string]string{"Tg-Chat-Id": "200"}
		resp = doRequest(t, client, "POST", scrapperBaseURL+"/links", addPayload, missingHeaders)
		require.NotEqual(t, http.StatusOK, resp.StatusCode, "Adding link to missing chat should fail")
		require.True(t, resp.StatusCode >= 400, "Should return 4xx error")
		resp.Body.Close()
	})

	t.Run("deleted chat", func(t *testing.T) {
		chatID := "105"
		resp := doRequest(t, client, "POST", scrapperBaseURL+"/tg-chat/"+chatID, nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Chat registration should succeed")
		resp.Body.Close()

		resp = doRequest(t, client, "DELETE", scrapperBaseURL+"/tg-chat/"+chatID, nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Deleting chat should return 200")
		resp.Body.Close()

		addPayload := map[string]interface{}{"link": "https://example.com/e"}
		headers := map[string]string{"Tg-Chat-Id": chatID}
		resp = doRequest(t, client, "POST", scrapperBaseURL+"/links", addPayload, headers)
		require.NotEqual(t, http.StatusOK, resp.StatusCode, "Adding link to deleted chat should fail")
		require.True(t, resp.StatusCode >= 400, "Should return 4xx error")
		resp.Body.Close()
	})

	t.Run("delete missing chat", func(t *testing.T) {
		resp := doRequest(t, client, "DELETE", scrapperBaseURL+"/tg-chat/999999", nil, nil)
		require.True(t, resp.StatusCode == http.StatusNotFound)
		resp.Body.Close()
	})
}
