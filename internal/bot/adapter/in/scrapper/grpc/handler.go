package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
	pb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/proto/bot"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedBotServiceServer
	notifier service.LinkUpdateNotifier
	logger   *slog.Logger
}

func NewHandler(notifier service.LinkUpdateNotifier, logger *slog.Logger) *Handler {
	return &Handler{
		notifier: notifier,
		logger:   logger,
	}
}

func (h *Handler) SendUpdate(_ context.Context, req *pb.SendUpdateRequest) (*pb.SendUpdateResponse, error) {
	h.logger.Info("grpc: received link update", slog.Int64("id", req.GetId()), slog.String("url", req.GetUrl()), slog.Int("chat_count", len(req.GetTgChatIds())))

	update := model.LinkUpdate{
		ID:          req.GetId(),
		URL:         req.GetUrl(),
		Description: req.GetDescription(),
		TgChatIDs:   req.GetTgChatIds(),
	}

	if err := h.notifier.Notify(update); err != nil {
		return nil, h.mapNotifyError(err)
	}

	return &pb.SendUpdateResponse{}, nil
}

func (h *Handler) mapNotifyError(err error) error {
	h.logger.Error("failed to notify chats from grpc update", slog.Any("error", err))

	switch {
	case errors.Is(err, model.ErrInvalidUpdate):
		return fmt.Errorf("grpc status: %w", status.Error(codes.InvalidArgument, err.Error()))
	case errors.Is(err, model.ErrDeliveryFailed):
		return fmt.Errorf("grpc status: %w", status.Error(codes.InvalidArgument, err.Error()))
	default:
		return fmt.Errorf("grpc status: %w", status.Error(codes.Internal, "failed to process update"))
	}
}
