# Тестовые запросы (curl)

Сервер: `go run ./cmd/api` (MongoDB: `docker compose up -d`). Команды для Windows PowerShell; в bash вместо `curl.exe` пишите `curl`. Тела POST/PATCH лежат в папке `requests/`. Идентификаторы в примерах — из реального прогона, подставьте свои из ответа `POST`.

| № | Сценарий | Ожидаемый код | Фактический код |
|---|----------|---------------|-----------------|
| 1 | Проверка здоровья | 200 | 200 |
| 2 | Создание заметки 1 | 201 | 201 |
| 3 | Создание заметки 2 (кириллица) | 201 | 201 |
| 4 | Создание заметки 3 | 201 | 201 |
| 5 | Дубликат title (уникальный индекс) | 409 | 409 |
| 6 | Создание без title | 400 | 400 |
| 7 | Список с поиском q=first, limit/skip | 200 | 200 |
| 8 | Получение по id | 200 | 200 |
| 9 | Частичное обновление (PATCH content) | 200 | 200 |
| 10 | PATCH: title уже занят | 409 | 409 |
| 11 | Некорректный id | 404 | 404 |
| 12 | Несуществующий id | 404 | 404 |
| 13 | Курсорная пагинация: страница 1 (limit=2) | 200 | 200 |
| 14 | Курсорная пагинация: страница 2 (after=...) | 200 | 200 |
| 15 | Текстовый поиск search=milk | 200 | 200 |
| 16 | Статистика (aggregation) | 200 | 200 |
| 17 | Заметка с TTL (ttlSeconds=5) | 201 | 201 |
| 18 | Удаление заметки 2 | 204 | 204 |
| 19 | Повторное удаление | 404 | 404 |
| 20 | TTL-индекс: заметка удалена автоматически (≈45 с после expiresAt) | 404 | 404 |

## 1. Проверка здоровья

```powershell
curl.exe -i http://localhost:8080/health
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json
{"status":"ok"}
```n
## 2. Создание заметки 1

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note1.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
{"id":"6abf80bcb0490fd86dfe7e73","title":"First note","content":"Hello Mongo!","createdAt":"2026-10-02T10:00:28.075Z","updatedAt":"2026-10-02T10:00:28.075Z"}
```n
## 3. Создание заметки 2 (кириллица)

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note2.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
{"id":"6abf80bcb0490fd86dfe7e74","title":"Вторая заметка","content":"Привет, документная база!","createdAt":"2026-10-02T10:00:28.333Z","updatedAt":"2026-10-02T10:00:28.333Z"}
```n
## 4. Создание заметки 3

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note3.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
{"id":"6abf80bcb0490fd86dfe7e75","title":"Shopping list","content":"milk, bread, eggs","createdAt":"2026-10-02T10:00:28.497Z","updatedAt":"2026-10-02T10:00:28.497Z"}
```n
## 5. Дубликат title (уникальный индекс)

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note1.json"
```n
Ответ:

```text
HTTP/1.1 409 Conflict
Content-Type: application/json; charset=utf-8
{"error":"title_already_exists"}
```n
## 6. Создание без title

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note_noTitle.json"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
{"error":"title_required"}
```n
## 7. Список с поиском q=first, limit/skip

```powershell
curl.exe -i "http://localhost:8080/api/v1/notes?limit=5&skip=0&q=first"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
[{"id":"6abf80bcb0490fd86dfe7e73","title":"First note","content":"Hello Mongo!","createdAt":"2026-10-02T10:00:28.075Z","updatedAt":"2026-10-02T10:00:28.075Z"}]
```n
## 8. Получение по id

```powershell
curl.exe -i http://localhost:8080/api/v1/notes/6abf80bcb0490fd86dfe7e73
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
{"id":"6abf80bcb0490fd86dfe7e73","title":"First note","content":"Hello Mongo!","createdAt":"2026-10-02T10:00:28.075Z","updatedAt":"2026-10-02T10:00:28.075Z"}
```n
## 9. Частичное обновление (PATCH content)

```powershell
curl.exe -i -X PATCH http://localhost:8080/api/v1/notes/6abf80bcb0490fd86dfe7e73 -H "Content-Type: application/json" -d "@requests/patch_content.json"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
{"id":"6abf80bcb0490fd86dfe7e73","title":"First note","content":"Updated content","createdAt":"2026-10-02T10:00:28.075Z","updatedAt":"2026-10-02T10:00:28.925Z"}
```n
## 10. PATCH: title уже занят

```powershell
curl.exe -i -X PATCH http://localhost:8080/api/v1/notes/6abf80bcb0490fd86dfe7e75 -H "Content-Type: application/json" -d "@requests/patch_title_dup.json"
```n
Ответ:

```text
HTTP/1.1 409 Conflict
Content-Type: application/json; charset=utf-8
{"error":"title_already_exists"}
```n
## 11. Некорректный id

```powershell
curl.exe -i http://localhost:8080/api/v1/notes/abc
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
{"error":"not_found"}
```n
## 12. Несуществующий id

```powershell
curl.exe -i http://localhost:8080/api/v1/notes/507f1f77bcf86cd799439011
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
{"error":"not_found"}
```n
## 13. Курсорная пагинация: страница 1 (limit=2)

```powershell
curl.exe -i "http://localhost:8080/api/v1/notes?limit=2"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Next-After: 6abf80bcb0490fd86dfe7e74
[{"id":"6abf80bcb0490fd86dfe7e75","title":"Shopping list","content":"milk, bread, eggs","createdAt":"2026-10-02T10:00:28.497Z","updatedAt":"2026-10-02T10:00:28.497Z"},{"id":"6abf80bcb0490fd86dfe7e74","title":"Вторая заметка","content":"Привет, документная база!","createdAt":"2026-10-02T10:00:28.333Z","updatedAt":"2026-10-02T10:00:28.333Z"}]
```n
## 14. Курсорная пагинация: страница 2 (after=...)

```powershell
curl.exe -i "http://localhost:8080/api/v1/notes?limit=2&after=6abf80bcb0490fd86dfe7e74"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
[{"id":"6abf80bcb0490fd86dfe7e73","title":"First note","content":"Updated content","createdAt":"2026-10-02T10:00:28.075Z","updatedAt":"2026-10-02T10:00:28.925Z"}]
```n
## 15. Текстовый поиск search=milk

```powershell
curl.exe -i "http://localhost:8080/api/v1/notes?search=milk"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
[{"id":"6abf80bcb0490fd86dfe7e75","title":"Shopping list","content":"milk, bread, eggs","createdAt":"2026-10-02T10:00:28.497Z","updatedAt":"2026-10-02T10:00:28.497Z"}]
```n
## 16. Статистика (aggregation)

```powershell
curl.exe -i http://localhost:8080/api/v1/notes/stats
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
{"count":3,"avgContentLength":19,"maxContentLength":25}
```n
## 17. Заметка с TTL (ttlSeconds=5)

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d "@requests/note_ttl.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
{"id":"6abf80bdb0490fd86dfe7e77","title":"Временная заметка","content":"удалится через 5 секунд","createdAt":"2026-10-02T10:00:29.635Z","updatedAt":"2026-10-02T10:00:29.635Z","expiresAt":"2026-10-02T10:00:34.635Z"}
```n
## 18. Удаление заметки 2

```powershell
curl.exe -i -X DELETE http://localhost:8080/api/v1/notes/6abf80bcb0490fd86dfe7e74
```n
Ответ:

```text
HTTP/1.1 204 No Content
```n
## 19. Повторное удаление

```powershell
curl.exe -i -X DELETE http://localhost:8080/api/v1/notes/6abf80bcb0490fd86dfe7e74
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
{"error":"not_found"}
```n
## 20. TTL-индекс: заметка удалена автоматически (≈45 с после expiresAt)

```powershell
curl.exe -i http://localhost:8080/api/v1/notes/6abf80bdb0490fd86dfe7e77
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
{"error":"not_found"}
```n
