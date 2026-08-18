package main

import (
	"context"
	"log"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/account"
	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/execution"
	"github.com/ModstDev/trading_platform/internal/httpapi"
	"github.com/ModstDev/trading_platform/internal/instrument"
	"github.com/ModstDev/trading_platform/internal/matching"
	"github.com/ModstDev/trading_platform/internal/order"
	"github.com/ModstDev/trading_platform/internal/position"
	"github.com/ModstDev/trading_platform/internal/pubsub"
	"github.com/ModstDev/trading_platform/internal/user"
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

	userService := user.NewService(db, queries)
	accountService := account.NewService(queries)
	instrumentService := instrument.NewService(queries)
	orderService := order.NewService(db, queries)
	positionService := position.NewService(queries)
	executionService := execution.NewService(queries)
	matchingService := matching.NewService(db)

	natsClient, err := pubsub.NewNATS("nats://localhost:4222")
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	ctx := context.Background()
	if err := natsClient.StartMatchingConsumer(ctx, matchingService); err != nil {
		log.Fatal(err)
	}

	server := httpapi.NewServer(userService, accountService, instrumentService, orderService, positionService, executionService, matchingService, cfg.JWT.Secret, natsClient)

	log.Println("API listening on :8080")

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}
