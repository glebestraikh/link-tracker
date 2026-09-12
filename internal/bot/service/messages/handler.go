package messages

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

const messagesScrapperTimeout = 5 * time.Second

type Handler struct {
	stateManager   service.StateManager
	scrapperClient service.ScrapperClient
	sender         service.ChatSender
	logger         *slog.Logger
}

func NewHandler(stateManager service.StateManager, scrapperClient service.ScrapperClient, sender service.ChatSender, logger *slog.Logger) *Handler {
	return &Handler{
		stateManager:   stateManager,
		scrapperClient: scrapperClient,
		sender:         sender,
		logger:         logger,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	state := h.stateManager.GetState(chatID, userID)
	text := strings.TrimSpace(msg.Text)

	switch state.Type {
	case model.StateWaitingURL:
		h.handleWaitingURL(chatID, userID, text)
	case model.StateWaitingTags:
		h.handleWaitingTags(ctx, chatID, userID, text, state.URL)
	case model.StateWaitingUntrackURL:
		h.handleWaitingUntrackURL(ctx, chatID, userID, text)
	default:
		return
	}
}

func (h *Handler) handleWaitingURL(chatID, userID int64, url string) {
	if !isValidURL(url) {
		h.logger.Info("invalid URL provided", slog.Int64("user_id", userID), slog.String("text", url))
		h.send(chatID, "Некорректная ссылка.")

		return
	}

	nextState := model.UserState{
		Type: model.StateWaitingTags,
		URL:  url,
	}
	h.stateManager.SetState(chatID, userID, nextState)

	h.logger.Info("URL accepted, waiting for tags", slog.Int64("user_id", userID), slog.String("url", url))
	h.send(chatID, "Ссылка принята. Введите теги через запятую (или «-» чтобы пропустить):")
}

func (h *Handler) handleWaitingTags(ctx context.Context, chatID, userID int64, text string, linkURL string) {
	var tags []string

	if text != "-" {
		for _, tag := range strings.Split(text, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	h.stateManager.ClearState(chatID, userID)

	ctx, cancel := context.WithTimeout(ctx, messagesScrapperTimeout)
	defer cancel()

	_, err := h.scrapperClient.AddLink(ctx, chatID, linkURL, tags)
	if err != nil {
		h.logger.Error("failed to add link", slog.Int64("user_id", userID), slog.String("url", linkURL), slog.Any("error", err))

		if errors.Is(err, model.ErrLinkAlreadyTracked) {
			h.send(chatID, fmt.Sprintf("Ошибка добавления ссылки: %s.", model.ErrLinkAlreadyTracked.Error()))
			return
		}

		h.send(chatID, model.ErrScrapperUnavailable.Error())

		return
	}

	h.logger.Info("link added successfully", slog.Int64("user_id", userID), slog.String("url", linkURL))
	h.send(chatID, fmt.Sprintf("Ссылка успешно добавлена: %s", linkURL))
}

func (h *Handler) handleWaitingUntrackURL(ctx context.Context, chatID, userID int64, url string) {
	if !isValidURL(url) {
		h.logger.Info("invalid URL provided for untrack", slog.Int64("user_id", userID), slog.String("text", url))
		h.send(chatID, "Некорректная ссылка.")

		return
	}

	h.stateManager.ClearState(chatID, userID)

	ctx, cancel := context.WithTimeout(ctx, messagesScrapperTimeout)
	defer cancel()

	_, err := h.scrapperClient.RemoveLink(ctx, chatID, url)
	if err != nil {
		h.logger.Error("failed to remove link", slog.Int64("user_id", userID), slog.String("url", url), slog.Any("error", err))

		if errors.Is(err, model.ErrLinkOrChatNotFound) {
			h.send(chatID, "Ошибка добавления ссылки: ссылка не найдена в отслеживаемых.")
			return
		}

		h.send(chatID, model.ErrScrapperUnavailable.Error())

		return
	}

	h.logger.Info("link removed successfully", slog.Int64("user_id", userID), slog.String("url", url))
	h.send(chatID, fmt.Sprintf("Ссылка удалена: %s", url))
}

func (h *Handler) send(chatID int64, text string) {
	if err := h.sender.Send(chatID, text); err != nil {
		h.logger.Error("failed to send message", slog.Int64("chat_id", chatID), slog.Any("error", err))
	}
}

func isValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())

	return strings.Contains(host, "github.com") || strings.Contains(host, "stackoverflow.com")
}
