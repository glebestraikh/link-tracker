package botgrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"
	pb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/proto/scrapper"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedScrapperServiceServer
	tracker service.Tracker
	logger  *slog.Logger
}

func NewHandler(tracker service.Tracker, logger *slog.Logger) *Handler {
	return &Handler{
		tracker: tracker,
		logger:  logger,
	}
}

func (h *Handler) RegisterChat(ctx context.Context, req *pb.RegisterChatRequest) (*pb.RegisterChatResponse, error) {
	h.logger.Info("grpc: register chat", slog.Int64("chat_id", req.GetChatId()))

	if err := h.tracker.RegisterChat(ctx, req.GetChatId()); err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.RegisterChatResponse{}, nil
}

func (h *Handler) DeleteChat(ctx context.Context, req *pb.DeleteChatRequest) (*pb.DeleteChatResponse, error) {
	h.logger.Info("grpc: delete chat", slog.Int64("chat_id", req.GetChatId()))

	if err := h.tracker.DeleteChat(ctx, req.GetChatId()); err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.DeleteChatResponse{}, nil
}

func (h *Handler) GetLinks(ctx context.Context, req *pb.GetLinksRequest) (*pb.GetLinksResponse, error) {
	h.logger.Info("grpc: get links", slog.Int64("chat_id", req.GetChatId()))

	links, err := h.tracker.GetLinks(ctx, req.GetChatId())
	if err != nil {
		return nil, h.mapServiceError(err)
	}

	pbLinks := make([]*pb.LinkResponse, 0, len(links))
	for _, l := range links {
		pbLinks = append(pbLinks, &pb.LinkResponse{
			Id:   l.ID,
			Url:  l.URL,
			Tags: l.Tags,
		})
	}

	return &pb.GetLinksResponse{
		Links: pbLinks,
		Size:  int32(len(links)),
	}, nil
}

func (h *Handler) AddLink(ctx context.Context, req *pb.AddLinkRequest) (*pb.LinkResponse, error) {
	h.logger.Info("grpc: add link", slog.Int64("chat_id", req.GetChatId()), slog.String("link", req.GetLink()))

	l, err := h.tracker.AddLink(ctx, req.GetChatId(), req.GetLink(), req.GetTags())
	if err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.LinkResponse{
		Id:   l.ID,
		Url:  l.URL,
		Tags: l.Tags,
	}, nil
}

func (h *Handler) RemoveLink(ctx context.Context, req *pb.RemoveLinkRequest) (*pb.LinkResponse, error) {
	h.logger.Info("grpc: remove link", slog.Int64("chat_id", req.GetChatId()), slog.String("link", req.GetLink()))

	l, err := h.tracker.RemoveLink(ctx, req.GetChatId(), req.GetLink())
	if err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.LinkResponse{
		Id:   l.ID,
		Url:  l.URL,
		Tags: l.Tags,
	}, nil
}

func (h *Handler) mapServiceError(err error) error {
	switch {
	case errors.Is(err, model.ErrChatAlreadyExists):
		return fmt.Errorf("grpc status: %w", status.Error(codes.AlreadyExists, err.Error()))
	case errors.Is(err, model.ErrLinkAlreadyTracked):
		return fmt.Errorf("grpc status: %w", status.Error(codes.AlreadyExists, err.Error()))
	case errors.Is(err, model.ErrChatNotFound):
		return fmt.Errorf("grpc status: %w", status.Error(codes.NotFound, err.Error()))
	case errors.Is(err, model.ErrLinkNotFound):
		return fmt.Errorf("grpc status: %w", status.Error(codes.NotFound, err.Error()))
	default:
		return fmt.Errorf("grpc status: %w", status.Error(codes.InvalidArgument, err.Error()))
	}
}
