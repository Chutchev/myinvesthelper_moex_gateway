package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/app"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.New(*cfg).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
