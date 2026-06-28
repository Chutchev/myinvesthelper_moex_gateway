package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/config"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/httpserver"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
)

const shutdownTimeout = 5 * time.Second

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	router := httpserver.NewRouter(moex.NewStubService(), cbr.NewStubService())
	return &App{
		server: &http.Server{
			Addr:              cfg.Server.Address(),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (a *App) Handler() http.Handler {
	return a.server.Handler
}

func (a *App) Run(ctx context.Context) error {
	serverDone := make(chan struct{})
	shutdownDone := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			shutdownDone <- a.server.Shutdown(shutdownCtx)
		case <-serverDone:
			shutdownDone <- nil
		}
	}()

	err := a.server.ListenAndServe()
	close(serverDone)
	shutdownErr := <-shutdownDone
	if shutdownErr != nil {
		if closeErr := a.server.Close(); closeErr != nil {
			return fmt.Errorf("shut down HTTP server: %w; force close: %v", shutdownErr, closeErr)
		}
		return fmt.Errorf("shut down HTTP server: %w", shutdownErr)
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
