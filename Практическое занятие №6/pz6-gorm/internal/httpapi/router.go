package httpapi

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func BuildRouter(d *gorm.DB) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)

	h := NewHandlers(d)

	r.Get("/health", h.Health)

	// Пользователи
	r.Post("/users", h.CreateUser)
	r.Get("/users/{id}", h.GetUserByID) // пользователь со всеми заметками и их тегами

	// Заметки
	r.Post("/notes", h.CreateNote)      // создаём заметку с тегами
	r.Get("/notes/{id}", h.GetNoteByID) // получаем заметку с автором и тегами

	return r
}
