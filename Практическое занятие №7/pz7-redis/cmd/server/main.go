package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Rudolf31/pz7-redis/internal/app"
	"github.com/Rudolf31/pz7-redis/internal/cache"
)

// Учебный DSN из ПЗ №5; для работы /tasks/{id} нужна запущенная БД todo.
const fallbackDSN = "postgres://postgres:postgres@127.0.0.1:5432/todo?sslmode=disable"

func main() {
	// 127.0.0.1, а не localhost: на Windows localhost сначала резолвится в IPv6 и тормозит подключение
	c := cache.New(envOr("REDIS_ADDR", "127.0.0.1:6379"))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := c.Ping(ctx); err != nil {
		log.Printf("WARN: redis is not reachable: %v", err)
	} else {
		log.Println("Connected to Redis")
	}
	cancel()

	h := &app.Handlers{Cache: c, DB: openDB(envOr("DATABASE_URL", fallbackDSN))}

	addr := ":" + envOr("PORT", "8080")
	log.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, h.Routes()))
}

// openDB подключается к PostgreSQL; при ошибке возвращает nil (маршрут /tasks/{id} отключается).
func openDB(dsn string) *sql.DB {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("WARN: postgres disabled: %v", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Printf("WARN: postgres disabled: %v", err)
		db.Close()
		return nil
	}
	log.Println("Connected to PostgreSQL")
	return db
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
