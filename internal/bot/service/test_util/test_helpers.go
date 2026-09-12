package testutil

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	TestMessageUserID   = 12345
	TestMessageUsername = "testuser"
)

func NewTestMessage(text string) *tgbotapi.Message {
	return &tgbotapi.Message{
		Text: text,
		From: &tgbotapi.User{ID: TestMessageUserID, UserName: TestMessageUsername},
		Chat: &tgbotapi.Chat{ID: TestMessageUserID},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Offset: 0,
				Length: len(strings.Fields(text)[0]),
			},
		},
	}
}
