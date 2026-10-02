# Тестовые запросы (curl)

Сервер: `go run .` (порт 8080 или `PORT`). Команды для Windows PowerShell; в bash вместо `curl.exe` пишите `curl`. Тела POST/PUT лежат в папке `requests/`. Для проверки сохранения задайте `DATA_FILE`.

| № | Запрос | Ожидаемый код | Фактический код |
|---|--------|---------------|-----------------|
| 1 | GET /health | 200 | 200 |
| 2 | POST /api/v1/tasks | 201 | 201 |
| 3 | POST /api/v1/tasks | 201 | 201 |
| 4 | POST /api/v1/tasks | 201 | 201 |
| 5 | GET /api/v1/tasks | 200 | 200 |
| 6 | GET /api/v1/tasks/1 | 200 | 200 |
| 7 | PUT /api/v1/tasks/1 | 200 | 200 |
| 8 | GET /api/v1/tasks?done=true | 200 | 200 |
| 9 | GET /api/v1/tasks?page=2&limit=2 | 200 | 200 |
| 10 | DELETE /api/v1/tasks/3 | 204 | 204 |
| 11 | GET /api/v1/tasks/3 | 404 | 404 |
| 12 | GET /api/v1/tasks/abc | 400 | 400 |
| 13 | POST /api/v1/tasks | 400 | 400 |
| 14 | POST /api/v1/tasks | 422 | 422 |
| 15 | PUT /api/v1/tasks/99 | 404 | 404 |
| 16 | GET /api/v1/tasks?limit=1000 | 400 | 400 |
| 17 | GET /api/tasks | 404 | 404 |
| 18 | GET /api/v1/tasks | 200 | 200 |

## 1. Проверка здоровья

```powershell
curl.exe -i "http://localhost:8080/health"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Length: 2
Content-Type: text/plain; charset=utf-8
OK
```n
## 2. Создание задачи 1

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "@requests/create1.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Content-Length: 149
{"id":1,"title":"Выучить chi","done":false,"created_at":"2026-10-02T10:57:39.3829721+03:00","updated_at":"2026-10-02T10:57:39.3829721+03:00"}
```n
## 3. Создание задачи 2

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "@requests/create2.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Content-Length: 156
{"id":2,"title":"Написать отчёт","done":false,"created_at":"2026-10-02T10:57:39.421601+03:00","updated_at":"2026-10-02T10:57:39.421601+03:00"}
```n
## 4. Создание задачи 3

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "@requests/create3.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Content-Length: 141
{"id":3,"title":"Third task","done":false,"created_at":"2026-10-02T10:57:39.4618406+03:00","updated_at":"2026-10-02T10:57:39.4618406+03:00"}
```n
## 5. Список задач

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Total-Count: 3
Content-Length: 448
[{"id":1,"title":"Выучить chi","done":false,"created_at":"2026-10-02T10:57:39.3829721+03:00","updated_at":"2026-10-02T10:57:39.3829721+03:00"},{"id":2,"title":"Написать отчёт","done":false,"created_at":"2026-10-02T10:57:39.421601+03:00","updated_at":"2026-10-02T10:57:39.421601+03:00"},{"id":3,"title":"Third task","done":false,"created_at":"2026-10-02T10:57:39.4618406+03:00","updated_at":"2026-10-02T10:57:39.4618406+03:00"}]
```n
## 6. Получение по id

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks/1"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 149
{"id":1,"title":"Выучить chi","done":false,"created_at":"2026-10-02T10:57:39.3829721+03:00","updated_at":"2026-10-02T10:57:39.3829721+03:00"}
```n
## 7. Обновление задачи 1

```powershell
curl.exe -i -X PUT http://localhost:8080/api/v1/tasks/1 -H "Content-Type: application/json" -d "@requests/update1.json"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 161
{"id":1,"title":"Выучить chi глубже","done":true,"created_at":"2026-10-02T10:57:39.3829721+03:00","updated_at":"2026-10-02T10:57:39.5912586+03:00"}
```n
## 8. Фильтр по done

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks?done=true"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Total-Count: 1
Content-Length: 163
[{"id":1,"title":"Выучить chi глубже","done":true,"created_at":"2026-10-02T10:57:39.3829721+03:00","updated_at":"2026-10-02T10:57:39.5912586+03:00"}]
```n
## 9. Пагинация (страница 2)

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks?page=2&limit=2"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Total-Count: 3
Content-Length: 143
[{"id":3,"title":"Third task","done":false,"created_at":"2026-10-02T10:57:39.4618406+03:00","updated_at":"2026-10-02T10:57:39.4618406+03:00"}]
```n
## 10. Удаление задачи 3

```powershell
curl.exe -i -X DELETE "http://localhost:8080/api/v1/tasks/3"
```n
Ответ:

```text
HTTP/1.1 204 No Content
```n
## 11. Получение удалённой задачи

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks/3"
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
Content-Length: 27
{"error":"task not found"}
```n
## 12. Некорректный id

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks/abc"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
Content-Length: 23
{"error":"invalid id"}
```n
## 13. Создание без title

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "@requests/empty.json"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
Content-Length: 30
{"error":"title is required"}
```n
## 14. Слишком короткий title

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "@requests/short.json"
```n
Ответ:

```text
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/json; charset=utf-8
Content-Length: 62
{"error":"title length must be between 3 and 100 characters"}
```n
## 15. Обновление несуществующей

```powershell
curl.exe -i -X PUT http://localhost:8080/api/v1/tasks/99 -H "Content-Type: application/json" -d "@requests/update99.json"
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
Content-Length: 27
{"error":"task not found"}
```n
## 16. Лимит больше максимума

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks?limit=1000"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
Content-Length: 64
{"error":"invalid limit: must be an integer between 1 and 100"}
```n
## 17. Маршрут без версии

```powershell
curl.exe -i "http://localhost:8080/api/tasks"
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Content-Length: 19
404 page not found
```n
## 18. Список после перезапуска сервера

```powershell
curl.exe -i "http://localhost:8080/api/v1/tasks"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Total-Count: 2
Content-Length: 319
[{"id":1,"title":"Выучить chi глубже","done":true,"created_at":"2026-10-02T10:57:39.3829721+03:00","updated_at":"2026-10-02T10:57:39.5912586+03:00"},{"id":2,"title":"Написать отчёт","done":false,"created_at":"2026-10-02T10:57:39.421601+03:00","updated_at":"2026-10-02T10:57:39.421601+03:00"}]
```n
