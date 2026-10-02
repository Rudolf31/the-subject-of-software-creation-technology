package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/Rudolf31/pz8-mongo/internal/db"
	"github.com/Rudolf31/pz8-mongo/internal/notes"
)

func main() {
	// .env не обязателен; если файла нет — ошибка игнорируется
	_ = godotenv.Load()

	uri := getenv("MONGO_URI", "mongodb://root:secret@127.0.0.1:27017/?authSource=admin")
	dbName := getenv("MONGO_DB", "pz8")
	addr := getenv("HTTP_ADDR", ":8080")

	deps, err := db.ConnectMongo(context.Background(), uri, dbName)
	if err != nil {
		log.Fatal("mongo connect:", err)
	}
	defer deps.Client.Disconnect(context.Background())
	log.Println("Connected to MongoDB, database", dbName)

	repo, err := notes.NewRepo(deps.Database)
	if err != nil {
		log.Fatal("notes repo:", err)
	}
	h := notes.NewHandler(repo)

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.Logger)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Mount("/api/v1/notes", h.Routes())

	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	log.Println("listening on", addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
