package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("API listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"status": "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
