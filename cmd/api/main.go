package main

import (
	"log"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/account"
	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/httpapi"
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

	server := httpapi.NewServer(userService, accountService, cfg.JWT.Secret)

	log.Println("API listening on :8080")

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}
