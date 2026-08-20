package main

import (
	"context"
	"log"

	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/outbox"
	"github.com/ModstDev/trading_platform/internal/pubsub"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	queries := database.New(db)

	natsClient, err := pubsub.NewNATS("nats://localhost:4222")
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	outboxService := outbox.NewService(
		queries,
		natsClient,
	)

	ctx := context.Background()

	log.Println("outbox worker started")

	outboxService.Run(ctx)
}
