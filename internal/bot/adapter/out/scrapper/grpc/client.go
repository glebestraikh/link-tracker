package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	pb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/proto/scrapper"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	client pb.ScrapperServiceClient
	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewClient(addr string, logger *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to scrapper gRPC: %w", err)
	}

	return &Client{
		client: pb.NewScrapperServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *Client) RegisterChat(ctx context.Context, chatID int64) error {
	_, err := c.client.RegisterChat(ctx, &pb.RegisterChatRequest{
		ChatId: chatID,
	})
	if err != nil {
		return c.mapError(err)
	}

	c.logger.Info("chat registered with scrapper via grpc", slog.Int64("chat_id", chatID))

	return nil
}

func (c *Client) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := c.client.DeleteChat(ctx, &pb.DeleteChatRequest{
		ChatId: chatID,
	})
	if err != nil {
		return c.mapError(err)
	}

	c.logger.Info("chat deleted in scrapper via grpc", slog.Int64("chat_id", chatID))

	return nil
}

func (c *Client) AddLink(ctx context.Context, chatID int64, url string, tags []string) (*model.Link, error) {
	resp, err := c.client.AddLink(ctx, &pb.AddLinkRequest{
		ChatId: chatID,
		Link:   url,
		Tags:   tags,
	})
	if err != nil {
		return nil, c.mapError(err)
	}

	return &model.Link{
		ID:   resp.GetId(),
		URL:  resp.GetUrl(),
		Tags: resp.GetTags(),
	}, nil
}

func (c *Client) RemoveLink(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	resp, err := c.client.RemoveLink(ctx, &pb.RemoveLinkRequest{
		ChatId: chatID,
		Link:   url,
	})
	if err != nil {
		return nil, c.mapError(err)
	}

	return &model.Link{
		ID:   resp.GetId(),
		URL:  resp.GetUrl(),
		Tags: resp.GetTags(),
	}, nil
}

func (c *Client) GetLinks(ctx context.Context, chatID int64) ([]*model.Link, error) {
	resp, err := c.client.GetLinks(ctx, &pb.GetLinksRequest{
		ChatId: chatID,
	})
	if err != nil {
		return nil, c.mapError(err)
	}

	links := make([]*model.Link, 0, len(resp.GetLinks()))
	for _, l := range resp.GetLinks() {
		links = append(links, &model.Link{
			ID:   l.GetId(),
			URL:  l.GetUrl(),
			Tags: l.GetTags(),
		})
	}

	return links, nil
}

func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close grpc connection: %w", err)
	}
	return nil
}

func (c *Client) mapError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return model.ErrScrapperUnavailable
	}

	switch st.Code() {
	case codes.AlreadyExists:
		msg := st.Message()
		if msg == model.ErrChatAlreadyRegistered.Error() {
			return model.ErrChatAlreadyRegistered
		}

		return model.ErrLinkAlreadyTracked
	case codes.NotFound:
		msg := st.Message()
		if msg == model.ErrChatNotFound.Error() {
			return model.ErrChatNotFound
		}

		return model.ErrLinkOrChatNotFound
	case codes.Unavailable:
		return model.ErrScrapperUnavailable
	case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument,
		codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented,
		codes.Internal, codes.DataLoss, codes.Unauthenticated:
		return fmt.Errorf("scrapper grpc error: %s", st.Message())
	}

	return fmt.Errorf("scrapper grpc error: %s", st.Message())
}
