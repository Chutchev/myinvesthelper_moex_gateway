package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cache"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/config"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/httpserver"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/logger"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
	"github.com/gofiber/fiber/v3"
)

const shutdownTimeout = 5 * time.Second

type App struct {
	server  *fiber.App
	address string
}

func New(cfg config.Config) *App {
	// Create logger
	log := logger.New(cfg.LogLevel)

	// Create cache
	cache := cache.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	// Create MOEX client and service
	moexClient := moex.NewHTTPClient(cfg.MOEXBaseURL, cfg.HTTPTimeout)
	moexService := moex.NewService(moexClient, cache, cfg.MarketCacheTTL)

	// Create CBR service (stub for now)
	cbrService := cbr.NewStubService()

	router := httpserver.NewRouter(moexService, cbrService, log)
	return &App{
		server:  router,
		address: cfg.Server.Address(),
	}
}

func (a *App) Handler() *fiber.App {
	return a.server
}

func (a *App) Run(ctx context.Context) error {
	err := a.server.Listen(a.address, fiber.ListenConfig{
		GracefulContext:       ctx,
		ShutdownTimeout:       shutdownTimeout,
		DisableStartupMessage: true,
	})
	if err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
