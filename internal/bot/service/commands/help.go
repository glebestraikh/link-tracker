package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type HelpCommand struct {
	commands []service.Command
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{}
}

func (c *HelpCommand) SetCommands(commands []service.Command) {
	c.commands = commands
}

func (c *HelpCommand) Name() string {
	return "help"
}

func (c *HelpCommand) Description() string {
	return "список доступных команд"
}

func (c *HelpCommand) Handle(_ context.Context, _ *tgbotapi.Message) string {
	var sb strings.Builder
	sb.WriteString("Доступные команды:\n")
	for _, cmd := range c.commands {
		fmt.Fprintf(&sb, "/%s — %s\n", cmd.Name(), cmd.Description())
	}

	return sb.String()
}
