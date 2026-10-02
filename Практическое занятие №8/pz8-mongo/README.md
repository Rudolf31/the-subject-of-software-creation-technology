# pz8-mongo

REST-сервис заметок на Go с MongoDB (практическое занятие №8): подключение официальным драйвером, коллекция `notes`, индексы, CRUD, фильтрация и пагинация, частичное обновление, обработка ошибок. Бонусы: полнотекстовый поиск, TTL-индекс, курсорная пагинация, агрегация.

## Требования

- Go ≥ 1.21 (проверено на 1.25)
- Docker и Docker Compose
- curl / Postman

## Запуск

```powershell
docker compose up -d                 # MongoDB 7 на 127.0.0.1:27017 (root / secret)
docker compose ps
Copy-Item .env.example .env          # bash: cp .env.example .env
go run ./cmd/api
```

Без `.env` используются значения по умолчанию (те же, что в `.env.example`). Адрес `127.0.0.1`, а не `localhost`, — чтобы не терять время на IPv6-резолв на Windows.

Консоль MongoDB:

```powershell
docker exec -it mongo-dev mongosh -u root -p secret --authenticationDatabase admin
```

Остановка: `docker compose down` (данные сохраняются в volume `mongo_data`; `docker compose down -v` — удалить и их).

## Модель и индексы

Документ в коллекции `notes` (создаётся автоматически при старте вместе с индексами):

```json
{ "_id": ObjectId("..."), "title": "...", "content": "...",
  "createdAt": ISODate("..."), "updatedAt": ISODate("..."), "expiresAt": ISODate("...") }
```

| Индекс               | Назначение                                                 |
|----------------------|------------------------------------------------------------|
| `title_unique`       | уникальный по `title` (дубликат → 409)                     |
| `title_content_text` | текстовый по `title` и `content` (`?search=`)              |
| `expiresAt_ttl`      | TTL, `expireAfterSeconds: 0`: удаление в момент `expiresAt` |

## Маршруты

| Метод и путь                       | Описание                                              | Коды               |
|------------------------------------|-------------------------------------------------------|--------------------|
| `GET /health`                      | `{"status":"ok"}`                                     | 200                |
| `POST /api/v1/notes`               | создать `{"title","content","ttlSeconds"?}`           | 201, 400, 409      |
| `GET /api/v1/notes`                | список: `q`, `search`, `limit`, `skip`, `after`       | 200, 400           |
| `GET /api/v1/notes/{id}`           | получить по id                                        | 200, 404           |
| `PATCH /api/v1/notes/{id}`         | частично обновить `{"title"?,"content"?}`             | 200, 400, 404, 409 |
| `DELETE /api/v1/notes/{id}`        | удалить                                               | 204, 404           |
| `GET /api/v1/notes/stats`          | количество заметок, средняя и максимальная длина `content` | 200           |

Список отсортирован от новых к старым (по `_id`). `q` — подстрока в заголовке без учёта регистра (спецсимволы как текст), `search` — полнотекстовый поиск по `title` и `content`, `limit` по умолчанию 20 (максимум 200). Курсорная пагинация: если страница полная, в заголовке `X-Next-After` приходит id для следующего запроса `?after=<id>`; `skip` и `after` вместе не используются. Некорректный `ObjectID` обрабатывается как «не найдено» (404). Ошибки — `{"error":"..."}`.

## Примеры запросов

Полный набор с ответами — в [requests.md](requests.md), тела запросов — в `requests/`.

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note1.json"
curl.exe -i "http://localhost:8080/api/v1/notes?limit=5&skip=0&q=first"
curl.exe -i http://localhost:8080/api/v1/notes/<id>
curl.exe -i -X PATCH http://localhost:8080/api/v1/notes/<id> -H "Content-Type: application/json" -d "@requests/patch_content.json"
curl.exe -i -X DELETE http://localhost:8080/api/v1/notes/<id>
```

Ожидаемо: `201` и документ с `id` при создании; `200` и заметка при чтении и обновлении (при PATCH меняются только переданные поля и `updatedAt`); `204` без тела при удалении.

## Тесты

```powershell
go test ./...
```

Интеграционные тесты создают отдельную БД `pz8_test_<время>` и удаляют её по окончании; если MongoDB недоступна — пропускаются. Адрес берётся из `MONGO_URI`.

## Типовые проблемы

| Проблема | Решение |
|----------|---------|
| `server selection error` / `connection refused` / i/o timeout | контейнер не запущен или порт 27017 не проброшен: `docker compose ps`, `docker compose up -d` |
| `Authentication failed` | в URI нужны `root:secret` и `?authSource=admin` |
| `409` / `duplicate key` | сработал уникальный индекс по `title` — выберите другой заголовок |
| `404` при существующем id | id должен быть 24-символьным hex-ObjectID |
| TTL-заметка не исчезла сразу | MongoDB проверяет TTL-индексы раз в 60 секунд — удаление происходит с задержкой до минуты |
| Кракозябры в консоли Windows | выполните `chcp 65001`; тела запросов с кириллицей передавайте из файла (`-d "@file.json"`) |

## Структура

```
pz8-mongo/
├─ cmd/api/main.go
├─ internal/db/mongo.go            # подключение, таймауты, ping
├─ internal/notes/model.go
├─ internal/notes/repo.go          # коллекция, индексы, CRUD, агрегация
├─ internal/notes/handler.go       # chi-обработчики
├─ internal/notes/repo_test.go
├─ internal/notes/handler_test.go
├─ requests/                       # тела запросов для curl
├─ requests.md
├─ docker-compose.yml
├─ .env.example
└─ go.mod
```
