# pz7-redis

Работа с Redis из Go (`go-redis/v9`): `SET`, `GET`, `TTL`, `DEL` и простой кэш «cache-aside» перед PostgreSQL (практическое занятие №7).

## Подготовка

```powershell
docker run --name pz7-redis -d -p 6379:6379 redis:latest
```

Для маршрута `/tasks/{id}` (необязательно) нужен PostgreSQL с таблицей `tasks` из ПЗ №5 (БД `todo`). Без неё сервер всё равно запустится, а `/tasks/{id}` вернёт 503.

## Запуск

```powershell
go run ./cmd/server
```

Переменные окружения (необязательные):

| Переменная     | По умолчанию                                                         |
|----------------|----------------------------------------------------------------------|
| `REDIS_ADDR`   | `127.0.0.1:6379`                                                     |
| `DATABASE_URL` | `postgres://postgres:postgres@127.0.0.1:5432/todo?sslmode=disable`   |
| `PORT`         | `8080`                                                               |

Используется `127.0.0.1`, а не `localhost`: на Windows `localhost` сначала резолвится в IPv6 и задерживает подключение к Docker.

## Маршруты

| Запрос                                      | Описание                                                  |
|---------------------------------------------|-----------------------------------------------------------|
| `GET /health`                               | проверка связи с Redis (`PING`)                           |
| `GET /set?key=k&value=v[&ttl=секунды]`      | `SET` с TTL (по умолчанию 10 с, максимум 3600)            |
| `GET /get?key=k`                            | `GET`; 404, если ключа нет или он истёк                   |
| `GET /ttl?key=k`                            | оставшееся время жизни; 404, если ключа нет               |
| `GET /del?key=k`                            | `DEL`                                                     |
| `GET /tasks/{id}`                           | задача из PostgreSQL через кэш, заголовок `X-Cache: HIT/MISS` |

```powershell
curl.exe -i "http://localhost:8080/set?key=test&value=hello"
curl.exe -i "http://localhost:8080/get?key=test"
curl.exe -i "http://localhost:8080/ttl?key=test"
```

Полный набор запросов с ответами — в [requests.md](requests.md).

## Как работает кэш (`/tasks/{id}`)

1. Ищем ключ `task:{id}` в Redis → при попадании отдаём (`X-Cache: HIT`).
2. При промахе читаем из PostgreSQL, кладём в Redis с TTL 30 с и отдаём (`X-Cache: MISS`).
3. Если Redis недоступен, запрос обслуживается из БД — кэш не должен ронять сервис.

## Тесты и бенчмарки

```powershell
go test ./...                                          # нужен Redis; тесты с БД пропускаются без PostgreSQL
go test ./internal/app -run xxx -bench . -benchtime 500x
```

## Структура

```
pz7-redis/
├─ cmd/server/main.go
├─ internal/cache/cache.go          # обёртка над go-redis: Set/Get/TTL/Del
├─ internal/cache/cache_test.go
├─ internal/app/handlers.go         # HTTP-маршруты, cache-aside
├─ internal/app/handlers_test.go
├─ requests.md
└─ go.mod
```
