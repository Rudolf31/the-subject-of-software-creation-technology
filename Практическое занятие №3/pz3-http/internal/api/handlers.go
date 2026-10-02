package api

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Rudolf31/pz3-http/internal/storage"
)

const (
	minTitleLen = 3
	maxTitleLen = 140
)

type Handlers struct {
	Store *storage.MemoryStore
}

func NewHandlers(store *storage.MemoryStore) *Handlers {
	return &Handlers{Store: store}
}

// Routes регистрирует все маршруты и возвращает готовый роутер.
func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /tasks", h.ListTasks)
	mux.HandleFunc("POST /tasks", h.CreateTask)
	mux.HandleFunc("GET /tasks/{id}", h.GetTask)
	mux.HandleFunc("PATCH /tasks/{id}", h.UpdateTask)
	mux.HandleFunc("DELETE /tasks/{id}", h.DeleteTask)
	return mux
}

// GET /health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /tasks, поддерживает фильтр ?q=text
func (h *Handlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := h.Store.List()

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if q != "" {
		filtered := make([]storage.Task, 0, len(tasks))
		for _, t := range tasks {
			if strings.Contains(strings.ToLower(t.Title), q) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	JSON(w, http.StatusOK, tasks)
}

type createTaskRequest struct {
	Title string `json:"title"`
}

// POST /tasks
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r) {
		BadRequest(w, "Content-Type must be application/json")
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid json: "+err.Error())
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		BadRequest(w, "title is required")
		return
	}
	if msg := validateTitle(req.Title); msg != "" {
		Unprocessable(w, msg)
		return
	}

	JSON(w, http.StatusCreated, h.Store.Create(req.Title))
}

// GET /tasks/{id}
func (h *Handlers) GetTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	t, err := h.Store.Get(id)
	if err != nil {
		storeError(w, err)
		return
	}
	JSON(w, http.StatusOK, t)
}

type updateTaskRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

// PATCH /tasks/{id}: меняет title и/или done
func (h *Handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if !isJSON(r) {
		BadRequest(w, "Content-Type must be application/json")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid json: "+err.Error())
		return
	}
	if req.Title == nil && req.Done == nil {
		BadRequest(w, "nothing to update: provide title and/or done")
		return
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if msg := validateTitle(trimmed); msg != "" {
			Unprocessable(w, msg)
			return
		}
		req.Title = &trimmed
	}

	t, err := h.Store.Update(id, req.Title, req.Done)
	if err != nil {
		storeError(w, err)
		return
	}
	JSON(w, http.StatusOK, t)
}

// DELETE /tasks/{id}
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.Store.Delete(id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isJSON допускает пустой Content-Type, иначе требует application/json.
func isJSON(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return true
	}
	mt, _, err := mime.ParseMediaType(ct)
	return err == nil && mt == "application/json"
}

func validateTitle(title string) string {
	n := utf8.RuneCountInString(title)
	if n < minTitleLen || n > maxTitleLen {
		return "title length must be between " + strconv.Itoa(minTitleLen) + " and " + strconv.Itoa(maxTitleLen) + " characters"
	}
	return ""
}

// parseID читает {id} из пути; при ошибке сам пишет ответ 400.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		BadRequest(w, "invalid id")
		return 0, false
	}
	return id, true
}

func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		NotFound(w, "task not found")
		return
	}
	Internal(w, "unexpected error")
}
