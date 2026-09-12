//go:build integration

package e2e_test

import (
	"context"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"io"
	"log"
	"os"
	"testing"
	"time"
)

const timeout = 5 * time.Minute

var (
	scrapperBaseURL string
	botBaseURL      string
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	repoRoot, err := projectRoot()
	if err != nil {
		log.Printf("failed to resolve repo root: %v", err)
		os.Exit(1)
	}

	net, err := network.New(ctx)
	if err != nil {
		log.Printf("failed to create network: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := net.Remove(ctx); err != nil {
			log.Printf("failed to remove network: %v", err)
		}
	}()

	log.Printf("Docker network created: %s", net.Name)

	postgresContainer, err := startPostgres(ctx, net)
	if err != nil {
		log.Printf("failed to start postgres: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("cleanup error (postgres container): %v", err)
		}
	}()

	log.Printf("Postgres started; bot+scrapper databases migrated")

	telegramContainer, err := startTelegramMock(ctx, repoRoot, net)
	if err != nil {
		log.Printf("failed to start telegram mock: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := telegramContainer.Terminate(ctx); err != nil {
			log.Printf("cleanup error (telegram container): %v", err)
		}
	}()

	go streamContainerLogs(ctx, "telegram-mock", telegramContainer)

	log.Printf("Telegram mock started")

	botContainer, botURL, err := startBot(ctx, repoRoot, net)
	if err != nil {
		log.Printf("failed to start bot: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := botContainer.Terminate(ctx); err != nil {
			log.Printf("cleanup error (bot container): %v", err)
		}
	}()

	botBaseURL = botURL

	go streamContainerLogs(ctx, "bot", botContainer)

	log.Printf("Bot started: %s", botBaseURL)

	scrapperContainer, scrapperURL, err := startScrapper(ctx, repoRoot, net)
	if err != nil {
		log.Printf("failed to start scrapper: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := scrapperContainer.Terminate(ctx); err != nil {
			log.Printf("cleanup error (scrapper container): %v", err)
		}
	}()

	scrapperBaseURL = scrapperURL

	go streamContainerLogs(ctx, "scrapper", scrapperContainer)

	log.Printf("Scrapper started: %s", scrapperBaseURL)

	log.Printf("\n=========================== Running integration tests ===========================")
	os.Exit(m.Run())
}

func streamContainerLogs(ctx context.Context, name string, c testcontainers.Container) {
	reader, err := c.Logs(ctx)
	if err != nil {
		log.Printf("failed to get logs for %s: %v", name, err)
		return
	}

	defer func() {
		err := reader.Close()
		if err != nil {
			log.Printf("failed to close logs: %v", err)
		}
	}()
	prefix := name + ": "
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			lines := buf[:n]
			os.Stderr.Write([]byte(prefix))
			os.Stderr.Write(lines)
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("error reading logs for %s: %v", name, err)
			}

			return
		}
	}
}
