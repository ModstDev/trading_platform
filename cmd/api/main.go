package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ModstDev/trading_platform/internal/account"
	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/execution"
	"github.com/ModstDev/trading_platform/internal/httpapi"
	"github.com/ModstDev/trading_platform/internal/instrument"
	"github.com/ModstDev/trading_platform/internal/marketdata"
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

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	natsClient, err := pubsub.NewNATS(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	priceStore := marketdata.NewPriceStore()

	priceSubscription, err := marketdata.StartConsumer(
		natsClient.Conn(),
		priceStore,
	)
	if err != nil {
		log.Fatalf("failed to start market data consumer: %v", err)
	}
	defer priceSubscription.Unsubscribe()

	server := httpapi.NewServer(
		userService,
		accountService,
		instrumentService,
		orderService,
		positionService,
		executionService,
		cfg.JWT.Secret,
		natsClient,
		priceStore,
	)

	log.Println("API listening on :8080")

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}
