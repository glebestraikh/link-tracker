package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Sender struct {
	api *tgbotapi.BotAPI
}

func NewSender(api *tgbotapi.BotAPI) *Sender {
	return &Sender{
		api: api,
	}
}

func (s *Sender) Send(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.api.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	return nil
}
