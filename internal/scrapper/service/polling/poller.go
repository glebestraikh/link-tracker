package polling

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"
)

const (
	pollPageSize = 100
	pollTimeout  = 30 * time.Second
)

type LinkPoller struct {
	linkRepo  service.LinkRepository
	checkers  []service.LinkChecker
	botClient service.BotClient
	scheduler gocron.Scheduler
	logger    *slog.Logger
	appCtx    context.Context
}

func NewLinkPoller(linkRepo service.LinkRepository, checkers []service.LinkChecker, botClient service.BotClient, interval time.Duration, logger *slog.Logger) (*LinkPoller, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create polling: %w", err)
	}

	poller := &LinkPoller{
		linkRepo:  linkRepo,
		checkers:  checkers,
		botClient: botClient,
		scheduler: scheduler,
		logger:    logger,
		appCtx:    context.Background(),
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(poller.checkLinks),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return poller, nil
}

func (p *LinkPoller) Start(ctx context.Context) {
	p.logger.Info("starting polling polling")
	p.appCtx = ctx
	p.scheduler.Start()
}

func (p *LinkPoller) Stop() {
	p.logger.Info("stopping polling polling")
	if err := p.scheduler.Shutdown(); err != nil {
		p.logger.Error("failed to shutdown polling", slog.Any("error", err))
	}
}

func (p *LinkPoller) checkLinks() {
	p.logger.Info("checking links for updates")

	ctx, cancel := context.WithTimeout(p.appCtx, pollTimeout)
	defer cancel()

	var offset uint64
	for {
		links, err := p.linkRepo.FindAllPaged(ctx, offset, pollPageSize)
		if err != nil {
			p.logger.Error("failed to get links", slog.Any("error", err))
			return
		}
		if len(links) == 0 {
			return
		}

		for _, link := range links {
			p.processLink(ctx, link)
		}

		if uint64(len(links)) < pollPageSize {
			return
		}
		offset += pollPageSize
	}
}

func (p *LinkPoller) processLink(ctx context.Context, link *model.Link) {
	for _, checker := range p.checkers {
		if !checker.Supports(link.URL) {
			continue
		}

		lastUpdated, checkerErr := checker.GetLastUpdated(ctx, link.URL)
		if checkerErr != nil {
			p.logger.Error("failed to check link", slog.String("url", link.URL), slog.Any("error", checkerErr))
			continue
		}

		if !lastUpdated.After(link.LastUpdated) {
			continue
		}

		p.logger.Info("link updated", slog.String("url", link.URL), slog.Time("last_updated", lastUpdated), slog.Time("previous", link.LastUpdated))

		if sendErr := p.botClient.SendUpdate(ctx, link.ID, link.URL, "Обнаружено обновление", link.ChatIDs); sendErr != nil {
			p.logger.Error("failed to send updates", slog.String("url", link.URL), slog.Any("error", sendErr))
			continue
		}

		if updateErr := p.linkRepo.UpdateLastUpdated(ctx, link.ID, lastUpdated); updateErr != nil {
			p.logger.Error("failed to updates last updated", slog.String("url", link.URL), slog.Any("error", updateErr))
		}
	}
}
