package app

import (
	"context"
	"testing"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/config"
	"github.com/gofiber/fiber/v3"
)

func TestRunStopsAfterContextCancellation(t *testing.T) {
	application := New(config.Config{Server: config.ServerConfig{Host: "127.0.0.1", Port: "0"}})
	var handler *fiber.App = application.Handler()
	listening := make(chan struct{})
	handler.Hooks().OnListen(func(fiber.ListenData) error {
		close(listening)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- application.Run(ctx) }()

	select {
	case <-listening:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start")
	}
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}
