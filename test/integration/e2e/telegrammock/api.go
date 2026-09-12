package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type apiResponse struct {
	OK     bool        `json:"ok"`
	Result interface{} `json:"result"`
}

type user struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type message struct {
	MessageID int64  `json:"message_id"`
	Chat      chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("telegram mock listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case strings.HasSuffix(r.URL.Path, "/getMe"):
		writeJSON(w, apiResponse{
			OK: true,
			Result: user{
				ID:        1,
				IsBot:     true,
				FirstName: "Test",
				Username:  "test_bot",
			},
		})

	case strings.HasSuffix(r.URL.Path, "/getUpdates"):
		writeJSON(w, apiResponse{
			OK:     true,
			Result: []interface{}{},
		})

	case strings.HasSuffix(r.URL.Path, "/sendMessage"):
		var req struct {
			ChatID int64  `json:"chat_id"`
			Text   string `json:"text"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Printf("failed to decode request body: %s", err)
		}

		writeJSON(w, apiResponse{
			OK: true,
			Result: message{
				MessageID: 1,
				Chat: chat{
					ID:   req.ChatID,
					Type: "private",
				},
				Text: req.Text,
				Date: time.Now().Unix(),
			},
		})

	default:
		writeJSON(w, apiResponse{
			OK:     true,
			Result: true,
		})
	}
}

func writeJSON(w http.ResponseWriter, payload apiResponse) {
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
