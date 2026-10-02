package notes

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	maxTitleLen   = 200
	maxTTLSeconds = 365 * 24 * 3600
	defaultLimit  = 20
	maxLimit      = 200
)

type Handler struct{ repo *Repo }

func NewHandler(r *Repo) *Handler { return &Handler{repo: r} }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/stats", h.stats) // статический маршрут имеет приоритет над /{id}
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.patch)
	r.Delete("/{id}", h.del)
	return r
}

func reqCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// fail переводит ошибки репозитория в HTTP-ответы; детали внутренних ошибок только в лог.
func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrDuplicate):
		writeErr(w, http.StatusConflict, "title_already_exists")
	case errors.Is(err, context.DeadlineExceeded):
		log.Printf("timeout: %v", err)
		writeErr(w, http.StatusGatewayTimeout, "database_timeout")
	default:
		log.Printf("internal error: %v", err)
		writeErr(w, http.StatusInternalServerError, "internal_error")
	}
}

func validTitle(title string) (string, string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "title_required"
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return "", "title_too_long"
	}
	return title, ""
}

// POST /api/v1/notes   {"title","content","ttlSeconds"?}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		TTLSeconds int64  `json:"ttlSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	title, msg := validTitle(in.Title)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if in.TTLSeconds < 0 || in.TTLSeconds > maxTTLSeconds {
		writeErr(w, http.StatusBadRequest, "invalid_ttlSeconds")
		return
	}

	c, cancel := reqCtx(r)
	defer cancel()
	n, err := h.repo.Create(c, title, in.Content, time.Duration(in.TTLSeconds)*time.Second)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

// GET /api/v1/notes/{id}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, cancel := reqCtx(r)
	defer cancel()
	n, err := h.repo.ByID(c, chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// GET /api/v1/notes?q=&search=&limit=&skip=&after=
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	p := ListParams{Q: qs.Get("q"), Search: qs.Get("search"), Limit: defaultLimit}

	if raw := qs.Get("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 || n > maxLimit {
			writeErr(w, http.StatusBadRequest, "invalid_limit")
			return
		}
		p.Limit = n
	}
	if raw := qs.Get("skip"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			writeErr(w, http.StatusBadRequest, "invalid_skip")
			return
		}
		p.Skip = n
	}
	if raw := qs.Get("after"); raw != "" {
		oid, err := primitive.ObjectIDFromHex(raw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_after")
			return
		}
		if p.Skip > 0 {
			writeErr(w, http.StatusBadRequest, "skip_and_after_are_exclusive")
			return
		}
		p.After = oid
	}

	c, cancel := reqCtx(r)
	defer cancel()
	items, err := h.repo.List(c, p)
	if err != nil {
		fail(w, err)
		return
	}
	// если страница полная, подсказываем курсор для следующего запроса: ?after=<X-Next-After>
	if int64(len(items)) == p.Limit {
		w.Header().Set("X-Next-After", items[len(items)-1].ID.Hex())
	}
	writeJSON(w, http.StatusOK, items)
}

// PATCH /api/v1/notes/{id}   {"title"?, "content"?}
func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if in.Title == nil && in.Content == nil {
		writeErr(w, http.StatusBadRequest, "nothing_to_update")
		return
	}
	if in.Title != nil {
		title, msg := validTitle(*in.Title)
		if msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		in.Title = &title
	}

	c, cancel := reqCtx(r)
	defer cancel()
	n, err := h.repo.Update(c, chi.URLParam(r, "id"), in.Title, in.Content)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// DELETE /api/v1/notes/{id}
func (h *Handler) del(w http.ResponseWriter, r *http.Request) {
	c, cancel := reqCtx(r)
	defer cancel()
	if err := h.repo.Delete(c, chi.URLParam(r, "id")); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/notes/stats
func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	c, cancel := reqCtx(r)
	defer cancel()
	s, err := h.repo.Stats(c)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}
