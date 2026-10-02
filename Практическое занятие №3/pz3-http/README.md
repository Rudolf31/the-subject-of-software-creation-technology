# pz3-http

REST-сервис задач на стандартной библиотеке `net/http` (практическое занятие №3). Хранилище — в памяти, сторонних зависимостей нет.

## Маршруты

| Метод и путь          | Описание                                   | Коды                  |
|-----------------------|--------------------------------------------|-----------------------|
| `GET /health`         | `{"status":"ok"}`                          | 200                   |
| `GET /tasks`          | список задач, фильтр `?q=text`             | 200                   |
| `POST /tasks`         | создать задачу `{"title":"..."}`           | 201, 400, 422         |
| `GET /tasks/{id}`     | одна задача                                | 200, 400, 404         |
| `PATCH /tasks/{id}`   | изменить `title` и/или `done`              | 200, 400, 404, 422    |
| `DELETE /tasks/{id}`  | удалить задачу                             | 204, 400, 404         |

Валидация: `title` обязателен (иначе 400), длина 3…140 символов (иначе 422). Ошибки всегда в виде `{"error":"..."}`.

Middleware (цепочка Recovery → Logging → CORS): перехват паник, лог метода/пути/кода/времени, CORS-заголовки. Сервер завершается корректно по Ctrl+C (graceful shutdown).

## Запуск

```powershell
go run ./cmd/server              # порт 8080
$env:PORT="9090"; go run ./cmd/server
```

Или через скрипт: `.\make.ps1 run | build | test`.

## Сборка и тесты

```powershell
go build -o .\bin\server.exe .\cmd\server
go test ./...
```

## Примеры запросов

Полный набор запросов с ответами — в [requests.md](requests.md), тела запросов — в папке `requests/`.

```powershell
curl.exe -i http://localhost:8080/health
curl.exe -i -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d "@requests/task1.json"
curl.exe -i http://localhost:8080/tasks
```

## Структура

```
pz3-http/
├─ cmd/server/main.go            # запуск, PORT, graceful shutdown
├─ internal/api/handlers.go      # обработчики и маршруты
├─ internal/api/handlers_test.go # тесты на httptest
├─ internal/api/middleware.go    # Logging, CORS, Recovery
├─ internal/api/responses.go     # JSON-ответы и ошибки
├─ internal/storage/memory.go    # in-memory хранилище
├─ requests/                     # тела запросов для curl
├─ requests.md
├─ make.ps1
└─ go.mod
```
