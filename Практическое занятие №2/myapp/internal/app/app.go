package app

import (
	"fmt"
	"net/http"

	"github.com/Rudolf31/myapp/internal/app/handlers"
	"github.com/Rudolf31/myapp/utils"
)

// withRequestID берёт X-Request-Id из запроса (или генерирует новый)
// и возвращает его в заголовке ответа.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = utils.NewID16()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}

func Run() {
	mux := http.NewServeMux()

	// Корневой маршрут
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		utils.LogRequest(r)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Hello, Go project structure!")
	})

	// JSON-ручка: /ping
	mux.HandleFunc("/ping", handlers.Ping)

	// Пример единого формата ошибок
	mux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		utils.LogRequest(r)
		utils.WriteErr(w, http.StatusBadRequest, "bad_request_example")
	})

	handler := withRequestID(mux)

	utils.LogInfo("Server is starting on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		utils.LogError("server error: " + err.Error())
	}
}
