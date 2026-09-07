// Command server запускает HTTP API сервиса QADAM.
package main

import (
	"log"
	"net/http"

	"github.com/qadam/backend/internal/config"
	"github.com/qadam/backend/internal/handlers"
	"github.com/qadam/backend/internal/middleware"
)

func main() {
	cfg := config.Load()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)

	var h http.Handler = mux
	h = middleware.Logging(h)

	addr := ":" + cfg.Port
	log.Printf("QADAM backend starting (env=%s) on %s", cfg.AppEnv, addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
