package commands

import (
	"context"
	"log/slog"
	"sort"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type Dispatcher struct {
	commands map[string]service.Command
	logger   *slog.Logger
}

func NewDispatcher(logger *slog.Logger, cmds ...service.Command) *Dispatcher {
	m := make(map[string]service.Command, len(cmds))
	for _, cmd := range cmds {
		m[cmd.Name()] = cmd
	}

	return &Dispatcher{commands: m, logger: logger}
}

func (d *Dispatcher) Dispatch(ctx context.Context, msg *tgbotapi.Message) string {
	cmd := msg.Command()

	d.logger.Info("received command", slog.String("command", cmd), slog.Int64("chat_id", msg.Chat.ID), slog.Int64("user_id", msg.From.ID))

	if handler, ok := d.commands[cmd]; ok {
		return handler.Handle(ctx, msg)
	}

	d.logger.Info("unknown command", slog.String("command", cmd), slog.Int64("chat_id", msg.Chat.ID), slog.Int64("user_id", msg.From.ID))

	return "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд."
}

func (d *Dispatcher) RegisteredCommands() []service.Command {
	cmds := make([]service.Command, 0, len(d.commands))
	for _, cmd := range d.commands {
		cmds = append(cmds, cmd)
	}

	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name() < cmds[j].Name()
	})

	return cmds
}
