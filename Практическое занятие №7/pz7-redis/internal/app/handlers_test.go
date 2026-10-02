package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Rudolf31/pz7-redis/internal/cache"
)

func redisAddr() string {
	if a := os.Getenv("REDIS_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:6379"
}

func newCache(t testing.TB) *cache.Cache {
	t.Helper()
	c := cache.New(redisAddr())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("Redis недоступен: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestSetGetTTLDelFlow(t *testing.T) {
	h := (&Handlers{Cache: newCache(t)}).Routes()
	key := fmt.Sprintf("flow%d", time.Now().UnixNano())
	t.Cleanup(func() { get(h, "/del?key="+key) })

	rec := get(h, "/set?key="+key+"&value=hello")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "TTL 10s") {
		t.Fatalf("set = %d %s", rec.Code, rec.Body.String())
	}
	if rec = get(h, "/get?key="+key); rec.Code != 200 || rec.Body.String() != "VALUE: "+key+"=hello" {
		t.Fatalf("get = %d %s", rec.Code, rec.Body.String())
	}
	// Redis отдаёт TTL с точностью до секунды: сразу после SET — 10s (или 9s)
	if rec = get(h, "/ttl?key="+key); rec.Code != 200 ||
		(rec.Body.String() != "TTL for "+key+": 10s" && rec.Body.String() != "TTL for "+key+": 9s") {
		t.Fatalf("ttl = %d %s", rec.Code, rec.Body.String())
	}
	if rec = get(h, "/del?key="+key); rec.Code != 200 {
		t.Fatalf("del = %d %s", rec.Code, rec.Body.String())
	}
	if rec = get(h, "/get?key="+key); rec.Code != http.StatusNotFound {
		t.Fatalf("get after del = %d", rec.Code)
	}
}

func TestCustomTTL(t *testing.T) {
	h := (&Handlers{Cache: newCache(t)}).Routes()
	key := fmt.Sprintf("ttl%d", time.Now().UnixNano())
	t.Cleanup(func() { get(h, "/del?key="+key) })

	if rec := get(h, "/set?key="+key+"&value=v&ttl=60"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "TTL 1m0s") {
		t.Fatalf("set = %d %s", rec.Code, rec.Body.String())
	}
}

func TestExpiredKeyIs404(t *testing.T) {
	h := (&Handlers{Cache: newCache(t)}).Routes()
	key := fmt.Sprintf("exp%d", time.Now().UnixNano())

	get(h, "/set?key="+key+"&value=v&ttl=1")
	time.Sleep(1200 * time.Millisecond)
	for _, path := range []string{"/get?key=" + key, "/ttl?key=" + key} {
		if rec := get(h, path); rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", path, rec.Code)
		}
	}
}

func TestBadRequests(t *testing.T) {
	h := (&Handlers{Cache: newCache(t)}).Routes()
	for _, path := range []string{
		"/set", "/set?key=a", "/set?value=a", "/set?key=a&value=b&ttl=0",
		"/set?key=a&value=b&ttl=abc", "/set?key=a&value=b&ttl=99999",
		"/get", "/ttl", "/del", "/tasks/abc",
	} {
		// /tasks/abc без БД вернёт 503, остальные — 400
		rec := get(h, path)
		want := http.StatusBadRequest
		if strings.HasPrefix(path, "/tasks") {
			want = http.StatusServiceUnavailable
		}
		if rec.Code != want {
			t.Errorf("%s = %d, want %d", path, rec.Code, want)
		}
	}
}

func TestHealthWithoutRedis(t *testing.T) {
	c := cache.New("127.0.0.1:1")
	defer c.Close()
	h := (&Handlers{Cache: c}).Routes()
	if rec := get(h, "/health"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("health = %d, want 503", rec.Code)
	}
}

// --- cache-aside для /tasks/{id}: нужна ещё и PostgreSQL ---

const defaultDSN = "postgres://postgres:postgres@127.0.0.1:5432/todo?sslmode=disable"

// newTaskServer создаёт временную схему с таблицей tasks и одной записью.
func newTaskServer(t testing.TB, c *cache.Cache) *Handlers {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDSN
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("PostgreSQL недоступен: %v", err)
	}
	if err := admin.Ping(); err != nil {
		t.Skipf("PostgreSQL недоступен: %v", err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE"); admin.Close() })

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := sql.Open("pgx", dsn+sep+"search_path="+schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE tasks (id SERIAL PRIMARY KEY, title TEXT NOT NULL,
		done BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
		INSERT INTO tasks (title) VALUES ('cached task')`)
	if err != nil {
		t.Fatal(err)
	}
	return &Handlers{Cache: c, DB: db}
}

func TestTaskCacheAside(t *testing.T) {
	c := newCache(t)
	hd := newTaskServer(t, c)
	h := hd.Routes()
	t.Cleanup(func() { c.Del(context.Background(), "task:1") })
	c.Del(context.Background(), "task:1")

	rec := get(h, "/tasks/1")
	if rec.Code != 200 || rec.Header().Get("X-Cache") != "MISS" || !strings.Contains(rec.Body.String(), "cached task") {
		t.Fatalf("first = %d X-Cache=%q %s", rec.Code, rec.Header().Get("X-Cache"), rec.Body.String())
	}
	// второй запрос — из Redis
	rec2 := get(h, "/tasks/1")
	if rec2.Code != 200 || rec2.Header().Get("X-Cache") != "HIT" || rec2.Body.String() != rec.Body.String() {
		t.Fatalf("second = %d X-Cache=%q", rec2.Code, rec2.Header().Get("X-Cache"))
	}
	// у ключа есть TTL
	if ttl, err := c.TTL(context.Background(), "task:1"); err != nil || ttl <= 0 || ttl > taskTTL {
		t.Errorf("ttl = %v, %v", ttl, err)
	}
	// после удаления ключа (инвалидация) снова MISS
	c.Del(context.Background(), "task:1")
	if rec3 := get(h, "/tasks/1"); rec3.Header().Get("X-Cache") != "MISS" {
		t.Errorf("after invalidation X-Cache=%q, want MISS", rec3.Header().Get("X-Cache"))
	}
}

func TestTaskNotFoundAndBadID(t *testing.T) {
	h := newTaskServer(t, newCache(t)).Routes()
	if rec := get(h, "/tasks/999"); rec.Code != http.StatusNotFound {
		t.Errorf("missing = %d, want 404", rec.Code)
	}
	if rec := get(h, "/tasks/abc"); rec.Code != http.StatusBadRequest {
		t.Errorf("abc = %d, want 400", rec.Code)
	}
}

// Если Redis недоступен, запрос всё равно обслуживается из БД.
func TestTaskServedFromDBWhenRedisDown(t *testing.T) {
	dead := cache.New("127.0.0.1:1")
	defer dead.Close()
	h := newTaskServer(t, newCache(t)) // заранее проверяем наличие БД и Redis
	h.Cache = dead
	rec := get(h.Routes(), "/tasks/1")
	if rec.Code != 200 || rec.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("= %d X-Cache=%q %s", rec.Code, rec.Header().Get("X-Cache"), rec.Body.String())
	}
}

func BenchmarkTaskFromDB(b *testing.B) {
	c := newCache(b)
	h := newTaskServer(b, c).Routes()
	ctx := context.Background()
	b.Cleanup(func() { c.Del(ctx, "task:1") })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c.Del(ctx, "task:1") // каждый раз промах
		b.StartTimer()
		get(h, "/tasks/1")
	}
}

func BenchmarkTaskFromCache(b *testing.B) {
	c := newCache(b)
	h := newTaskServer(b, c).Routes()
	b.Cleanup(func() { c.Del(context.Background(), "task:1") })
	get(h, "/tasks/1") // прогрев кэша
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		get(h, "/tasks/1")
	}
}
