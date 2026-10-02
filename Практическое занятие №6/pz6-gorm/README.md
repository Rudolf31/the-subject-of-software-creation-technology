# pz6-gorm

REST-сервис «Пользователи — Заметки — Теги» на [GORM](https://gorm.io) и PostgreSQL (практическое занятие №6): модели, автомиграции `AutoMigrate`, связи 1:N и M:N, `Preload`.

## Модели и связи

- `User` 1 ── N `Note` (`notes.user_id`, внешний ключ)
- `Note` N ── M `Tag` (таблица `note_tags`, тег `many2many:note_tags`)
- `users.email` и `tags.name` — уникальные индексы

## Подготовка БД

PostgreSQL 14+ (в этой работе — контейнер из ПЗ №5):

```powershell
docker run -d --name pz5-postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:latest
docker exec pz5-postgres psql -U postgres -c "CREATE DATABASE pz6_gorm;"
```

Таблицы создаются сами при старте сервера (`AutoMigrate`).

## Настройка и запуск

DSN задаётся переменной окружения `DB_DSN`:

```powershell
$env:DB_DSN="host=127.0.0.1 user=postgres password=postgres dbname=pz6_gorm port=5432 sslmode=disable"
go run ./cmd/server
```

macOS/Linux: `export DB_DSN='host=127.0.0.1 user=postgres password=postgres dbname=pz6_gorm port=5432 sslmode=disable'`

Необязательные переменные: `PORT` (по умолчанию 8080), `GORM_LOG=info` — печатать весь SQL, который генерирует GORM.

Адрес `127.0.0.1` вместо `localhost` — чтобы не терять около секунды на каждое подключение из-за IPv6 (см. ПЗ №5).

## Маршруты

| Метод и путь        | Описание                                         | Коды               |
|---------------------|--------------------------------------------------|--------------------|
| `GET /health`       | `{"status":"ok"}`                                | 200                |
| `POST /users`       | `{"name","email"}`                               | 201, 400, 409      |
| `GET /users/{id}`   | пользователь с заметками и их тегами             | 200, 400, 404      |
| `POST /notes`       | `{"title","content","userId","tags":[...]}`      | 201, 400, 422      |
| `GET /notes/{id}`   | заметка с автором и тегами                       | 200, 400, 404      |

Заметка и её теги создаются в одной транзакции; существующие теги переиспользуются. Ошибки — `{"error":"..."}`.

## Примеры запросов

Полный набор с ответами — в [requests.md](requests.md); тела запросов — в `requests/`.

```powershell
curl.exe -i http://localhost:8080/health
curl.exe -i -X POST http://localhost:8080/users -H "Content-Type: application/json" -d "@requests/user1.json"
curl.exe -i -X POST http://localhost:8080/notes -H "Content-Type: application/json" -d "@requests/note1.json"
curl.exe -i http://localhost:8080/notes/1
```

## Тесты

```powershell
go test ./...
```

Интеграционные тесты используют отдельную временную схему PostgreSQL и удаляют её после прогона; если БД недоступна — пропускаются.

## Структура

```
pz6-gorm/
├─ cmd/server/main.go               # подключение, AutoMigrate, запуск
├─ internal/db/postgres.go          # GORM + настройка пула
├─ internal/models/models.go        # User, Note, Tag
├─ internal/httpapi/router.go       # chi-маршруты
├─ internal/httpapi/handlers.go     # обработчики
├─ internal/httpapi/handlers_test.go
├─ requests/                        # тела запросов
├─ requests.md
└─ go.mod
```
