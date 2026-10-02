# pz4-todo

CRUD-сервис «Список задач» на роутере [chi](https://github.com/go-chi/chi) (практическое занятие №4). Хранилище — память с необязательным сохранением в JSON-файл.

## Маршруты

| Метод и путь             | Описание                                | Коды                 |
|--------------------------|-----------------------------------------|----------------------|
| `GET /health`            | `OK`                                    | 200                  |
| `GET /api/v1/tasks`      | список; `?done=true`, `?page=1&limit=10`| 200, 400             |
| `POST /api/v1/tasks`     | создать `{"title":"..."}`               | 201, 400, 415, 422   |
| `GET /api/v1/tasks/{id}` | одна задача                             | 200, 400, 404        |
| `PUT /api/v1/tasks/{id}` | заменить `{"title":"...","done":bool}`  | 200, 400, 404, 422   |
| `DELETE /api/v1/tasks/{id}` | удалить                              | 204, 400, 404        |

- `title`: обязателен (иначе 400), длина 3…100 символов (иначе 422).
- Пагинация: `limit` по умолчанию 10, максимум 100; общее число подходящих задач — в заголовке `X-Total-Count`. Без `page`/`limit` возвращаются все задачи.
- Ошибки всегда в виде `{"error":"..."}`.
- Middleware: `RequestID`, `Recoverer` (chi), `Logger` (метод, путь, код, время), `SimpleCORS`.

## Запуск

```powershell
go run .                                  # порт 8080, данные только в памяти
$env:PORT="9090"; go run .                # другой порт
$env:DATA_FILE="tasks.json"; go run .     # сохранять задачи в файл
```

## Сборка и тесты

```powershell
go build -o bin\todo.exe .
go test ./...
```

## Примеры запросов

Полный набор с ответами — в [requests.md](requests.md); тела запросов — в `requests/`.

```powershell
curl.exe -i http://localhost:8080/health
curl.exe -i -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "@requests/create1.json"
curl.exe -i "http://localhost:8080/api/v1/tasks?done=true&page=1&limit=10"
```

## Структура

```
pz4-todo/
├─ main.go                    # роутер, middleware, запуск
├─ main_test.go               # тесты API (httptest)
├─ internal/task/model.go     # модель Task
├─ internal/task/repo.go      # хранилище (память + JSON-файл)
├─ internal/task/handler.go   # CRUD-обработчики и chi-маршруты
├─ pkg/middleware/logger.go
├─ pkg/middleware/cors.go
├─ requests/                  # тела запросов для curl
├─ requests.md
└─ go.mod
```
