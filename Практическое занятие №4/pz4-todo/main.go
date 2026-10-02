package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Rudolf31/pz4-todo/internal/task"
	myMW "github.com/Rudolf31/pz4-todo/pkg/middleware"
)

func newRouter(repo *task.Repo) chi.Router {
	h := task.NewHandler(repo)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(myMW.Logger)
	r.Use(myMW.SimpleCORS)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/tasks", h.Routes())
	})
	return r
}

func main() {
	addr := ":" + envOr("PORT", "8080")

	// DATA_FILE включает сохранение задач в JSON-файл (по умолчанию — только память).
	repo, err := task.NewRepo(os.Getenv("DATA_FILE"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newRouter(repo)))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
