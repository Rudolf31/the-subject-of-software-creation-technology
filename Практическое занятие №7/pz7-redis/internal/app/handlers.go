package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Rudolf31/pz7-redis/internal/cache"
)

const (
	defaultTTL = 10 * time.Second // TTL по умолчанию для /set
	maxTTL     = time.Hour
	taskTTL    = 30 * time.Second // TTL кэша задач
	reqTimeout = 3 * time.Second
)

// Task — запись из таблицы tasks (ПЗ №5), которую кэшируем.
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

type Handlers struct {
	Cache *cache.Cache
	DB    *sql.DB // может быть nil — тогда /tasks/{id} недоступен
}

func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /set", h.Set)
	mux.HandleFunc("GET /get", h.Get)
	mux.HandleFunc("GET /ttl", h.TTL)
	mux.HandleFunc("GET /del", h.Del)
	mux.HandleFunc("GET /tasks/{id}", h.GetTask)
	return mux
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()
	if err := h.Cache.Ping(ctx); err != nil {
		http.Error(w, "redis unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "OK")
}

// GET /set?key=..&value=..[&ttl=секунды] — сохранить значение с TTL (по умолчанию 10 с)
func (h *Handlers) Set(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key, value := q.Get("key"), q.Get("value")
	if key == "" || value == "" {
		http.Error(w, "key and value required", http.StatusBadRequest)
		return
	}
	ttl := defaultTTL
	if raw := q.Get("ttl"); raw != "" {
		sec, err := strconv.Atoi(raw)
		if err != nil || sec < 1 || time.Duration(sec)*time.Second > maxTTL {
			http.Error(w, "ttl must be an integer between 1 and 3600 seconds", http.StatusBadRequest)
			return
		}
		ttl = time.Duration(sec) * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()
	if err := h.Cache.Set(ctx, key, value, ttl); err != nil {
		internalErr(w, err)
		return
	}
	fmt.Fprintf(w, "OK: %s=%s (TTL %s)", key, value, ttl)
}

// GET /get?key=..
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()
	val, err := h.Cache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		// ключа нет или он «протух» по TTL
		http.Error(w, "key not found (missing or expired)", http.StatusNotFound)
		return
	}
	if err != nil {
		internalErr(w, err)
		return
	}
	fmt.Fprintf(w, "VALUE: %s=%s", key, val)
}

// GET /ttl?key=..
func (h *Handlers) TTL(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()
	ttl, err := h.Cache.TTL(ctx, key)
	switch {
	case errors.Is(err, cache.ErrNotFound):
		http.Error(w, "key not found (missing or expired)", http.StatusNotFound)
	case err != nil:
		internalErr(w, err)
	case ttl == cache.NoExpiry:
		fmt.Fprintf(w, "TTL for %s: no expiry", key)
	default:
		fmt.Fprintf(w, "TTL for %s: %v", key, ttl.Round(time.Millisecond))
	}
}

// GET /del?key=..
func (h *Handlers) Del(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()
	deleted, err := h.Cache.Del(ctx, key)
	if err != nil {
		internalErr(w, err)
		return
	}
	if !deleted {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "DELETED: %s", key)
}

// GET /tasks/{id} — кэш «cache-aside»: сначала Redis, при промахе — PostgreSQL,
// результат кладётся в Redis на taskTTL. Заголовок X-Cache: HIT или MISS.
func (h *Handlers) GetTask(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		http.Error(w, "database is not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()
	key := "task:" + strconv.Itoa(id)

	// 1) пробуем кэш; ошибка Redis не должна ронять запрос — идём в БД
	switch val, err := h.Cache.Get(ctx, key); {
	case err == nil:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Cache", "HIT")
		fmt.Fprintln(w, val)
		return
	case !errors.Is(err, cache.ErrNotFound):
		log.Printf("cache get %s: %v (fallback to DB)", key, err)
	}

	// 2) промах — читаем из PostgreSQL
	var t Task
	err = h.DB.QueryRowContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if err != nil {
		internalErr(w, err)
		return
	}

	// 3) кладём в кэш с TTL
	body, _ := json.Marshal(t)
	if err := h.Cache.Set(ctx, key, string(body), taskTTL); err != nil {
		log.Printf("cache set %s: %v", key, err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Cache", "MISS")
	fmt.Fprintln(w, string(body))
}

// internalErr пишет подробности в лог, клиенту отдаёт общее сообщение.
func internalErr(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
