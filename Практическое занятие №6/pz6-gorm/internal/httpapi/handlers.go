package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/Rudolf31/pz6-gorm/internal/models"
)

type Handlers struct{ db *gorm.DB }

func NewHandlers(db *gorm.DB) *Handlers { return &Handlers{db: db} }

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// POST /users
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in createUserReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Name, in.Email = strings.TrimSpace(in.Name), strings.TrimSpace(in.Email)
	if in.Name == "" || in.Email == "" {
		writeErr(w, http.StatusBadRequest, "name and email are required")
		return
	}

	u := models.User{Name: in.Name, Email: in.Email}
	if err := h.db.Create(&u).Error; err != nil {
		// TranslateError превращает нарушение uniqueIndex в gorm.ErrDuplicatedKey
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			writeErr(w, http.StatusConflict, "user with this email already exists")
			return
		}
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// GET /users/{id} — пользователь с заметками (1:N) и тегами заметок (M:N)
func (h *Handlers) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var user models.User
	if err := h.db.Preload("Notes.Tags").First(&user, id).Error; err != nil {
		dbErr(w, err, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type createNoteReq struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	UserID  uint     `json:"userId"`
	Tags    []string `json:"tags"` // имена тегов
}

// POST /notes — заметка и её теги создаются в одной транзакции
func (h *Handlers) CreateNote(w http.ResponseWriter, r *http.Request) {
	var in createNoteReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || in.UserID == 0 {
		writeErr(w, http.StatusBadRequest, "title and userId are required")
		return
	}

	var note models.Note
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// автор должен существовать
		if err := tx.Select("id").First(&models.User{}, in.UserID).Error; err != nil {
			return err
		}

		// Находим/создаём теги (без дубликатов и пустых имён)
		var tags []models.Tag
		seen := map[string]bool{}
		for _, name := range in.Tags {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			t := models.Tag{Name: name}
			if err := tx.FirstOrCreate(&t, models.Tag{Name: name}).Error; err != nil {
				return err
			}
			tags = append(tags, t)
		}

		note = models.Note{Title: in.Title, Content: in.Content, UserID: in.UserID, Tags: tags}
		return tx.Create(&note).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeErr(w, http.StatusUnprocessableEntity, "user not found")
			return
		}
		serverErr(w, err)
		return
	}

	// Вернём с автором и тегами
	if err := h.db.Preload("User").Preload("Tags").First(&note, note.ID).Error; err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

// GET /notes/{id}
func (h *Handlers) GetNoteByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var note models.Note
	if err := h.db.Preload("User").Preload("Tags").First(&note, id).Error; err != nil {
		dbErr(w, err, "note not found")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// helpers

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "bad id")
		return 0, false
	}
	return id, true
}

// dbErr: «не найдено» → 404, всё остальное → 500.
func dbErr(w http.ResponseWriter, err error, notFoundMsg string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeErr(w, http.StatusNotFound, notFoundMsg)
		return
	}
	serverErr(w, err)
}

// serverErr пишет подробности в лог, а клиенту отдаёт общее сообщение.
func serverErr(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

type jsonErr struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, jsonErr{Error: msg})
}
