package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

const listScrapperTimeout = 5 * time.Second

type ListCommand struct {
	scrapperClient service.ScrapperClient
	logger         *slog.Logger
}

func NewListCommand(scrapperClient service.ScrapperClient, logger *slog.Logger) *ListCommand {
	return &ListCommand{scrapperClient: scrapperClient, logger: logger}
}

func (c *ListCommand) Name() string {
	return "list"
}

func (c *ListCommand) Description() string {
	return "список отслеживаемых ссылок"
}

func (c *ListCommand) Handle(ctx context.Context, msg *tgbotapi.Message) string {
	chatID := msg.Chat.ID
	c.logger.Info("list command received", slog.Int64("chat_id", chatID), slog.String("command", "list"))

	ctx, cancel := context.WithTimeout(ctx, listScrapperTimeout)
	defer cancel()

	links, err := c.scrapperClient.GetLinks(ctx, chatID)
	if err != nil {
		c.logger.Error("failed to get links", slog.Int64("chat_id", chatID), slog.String("error", err.Error()))

		return model.ErrScrapperUnavailable.Error()
	}

	tagFilter := strings.TrimSpace(msg.CommandArguments())

	filtered := c.filterLinksByTag(links, tagFilter)

	if len(filtered) == 0 {
		if tagFilter != "" {
			return "Нет ссылок с таким тегом."
		}

		return "Нет отслеживаемых ссылок."
	}

	return c.formatLinksList(filtered)
}

func (c *ListCommand) filterLinksByTag(links []*model.Link, tagFilter string) []*model.Link {
	if tagFilter == "" {
		return links
	}

	var filtered []*model.Link
	for _, link := range links {
		for _, tag := range link.Tags {
			if tag == tagFilter {
				filtered = append(filtered, link)
				break
			}
		}
	}

	return filtered
}

func (c *ListCommand) formatLinksList(links []*model.Link) string {
	var sb strings.Builder
	sb.WriteString("Отслеживаемые ссылки:\n")
	for i, link := range links {
		if len(link.Tags) > 0 {
			fmt.Fprintf(&sb, "%d. %s [теги: %s]\n", i+1, link.URL, strings.Join(link.Tags, ", "))
		} else {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, link.URL)
		}
	}

	return sb.String()
}
