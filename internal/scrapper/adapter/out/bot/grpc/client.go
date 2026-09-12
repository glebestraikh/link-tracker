package grpc

import (
	"context"
	"fmt"
	"log/slog"

	pb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/proto/bot"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client pb.BotServiceClient
	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewClient(addr string, logger *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bot gRPC: %w", err)
	}

	return &Client{
		client: pb.NewBotServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *Client) SendUpdate(ctx context.Context, id int64, linkURL string, description string, tgChatIDs []int64) error {
	_, err := c.client.SendUpdate(ctx, &pb.SendUpdateRequest{
		Id:          id,
		Url:         linkURL,
		Description: description,
		TgChatIds:   tgChatIDs,
	})
	if err != nil {
		return fmt.Errorf("grpc send update failed: %w", err)
	}

	c.logger.Info("update sent to bot via grpc", slog.Int64("link_id", id), slog.String("url", linkURL))

	return nil
}

func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close grpc connection: %w", err)
	}
	return nil
}
