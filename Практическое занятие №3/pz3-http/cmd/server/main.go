package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Rudolf31/pz3-http/internal/api"
	"github.com/Rudolf31/pz3-http/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := storage.NewMemoryStore()
	h := api.NewHandlers(store)

	// Цепочка: Recovery -> Logging -> CORS -> маршруты
	handler := api.Recovery(api.Logging(api.CORS(h.Routes())))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown по Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		log.Println("listening on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
	log.Println("stopped")
}
