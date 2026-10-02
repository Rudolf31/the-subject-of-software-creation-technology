package task

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

const (
	minTitleLen = 3
	maxTitleLen = 100
	maxLimit    = 100
)

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)          // GET /tasks
	r.Post("/", h.create)       // POST /tasks
	r.Get("/{id}", h.get)       // GET /tasks/{id}
	r.Put("/{id}", h.update)    // PUT /tasks/{id}
	r.Delete("/{id}", h.delete) // DELETE /tasks/{id}
	return r
}

// GET /tasks?done=true&page=1&limit=10
//
// Фильтр done и пагинация необязательны; без page/limit возвращаются все задачи.
// Общее число подходящих задач передаётся в заголовке X-Total-Count.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tasks := h.repo.List()

	if raw := q.Get("done"); raw != "" {
		done, err := strconv.ParseBool(raw)
		if err != nil {
			httpError(w, http.StatusBadRequest, "invalid done: use true or false")
			return
		}
		filtered := make([]Task, 0, len(tasks))
		for _, t := range tasks {
			if t.Done == done {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	total := len(tasks)
	if q.Has("page") || q.Has("limit") {
		page, ok := intParam(w, q.Get("page"), "page", 1, 1<<30)
		if !ok {
			return
		}
		limit, ok := intParam(w, q.Get("limit"), "limit", 10, maxLimit)
		if !ok {
			return
		}
		start := (page - 1) * limit
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		tasks = tasks[start:end]
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.repo.Get(id)
	if err != nil {
		repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type createReq struct {
	Title string `json:"title"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if !decodeBody(w, r, &req) {
		return
	}
	title, ok := validTitle(w, req.Title)
	if !ok {
		return
	}
	t, err := h.repo.Create(title)
	if err != nil {
		repoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

type updateReq struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req updateReq
	if !decodeBody(w, r, &req) {
		return
	}
	title, ok := validTitle(w, req.Title)
	if !ok {
		return
	}
	t, err := h.repo.Update(id, title, req.Done)
	if err != nil {
		repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Delete(id); err != nil {
		repoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// helpers

// decodeBody проверяет Content-Type и разбирает JSON; при ошибке пишет 400/415.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		httpError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

// validTitle обрезает пробелы и проверяет длину (3..100 символов).
func validTitle(w http.ResponseWriter, raw string) (string, bool) {
	title := strings.TrimSpace(raw)
	if title == "" {
		httpError(w, http.StatusBadRequest, "title is required")
		return "", false
	}
	if n := utf8.RuneCountInString(title); n < minTitleLen || n > maxTitleLen {
		httpError(w, http.StatusUnprocessableEntity,
			"title length must be between "+strconv.Itoa(minTitleLen)+" and "+strconv.Itoa(maxTitleLen)+" characters")
		return "", false
	}
	return title, true
}

// intParam читает положительный целочисленный query-параметр; пустое значение — def.
func intParam(w http.ResponseWriter, raw, name string, def, max int) (int, bool) {
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		httpError(w, http.StatusBadRequest, "invalid "+name+": must be an integer between 1 and "+strconv.Itoa(max))
		return 0, false
	}
	return n, true
}

// parseID читает {id} из пути; при ошибке сам пишет ответ 400 и возвращает false.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func repoError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		httpError(w, http.StatusNotFound, err.Error())
		return
	}
	log.Printf("repo error: %v", err)
	httpError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
