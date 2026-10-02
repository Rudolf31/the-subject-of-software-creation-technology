# Тестовые запросы (curl)

Сервер: `go run ./cmd/server` (порт 8080, либо `PORT`). Команды для Windows PowerShell; в bash вместо `curl.exe` пишите `curl`. Тела POST/PATCH лежат в папке `requests/`.

## 1. GET /health

```powershell
curl.exe -i http://localhost:8080/health
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 16
{"status":"ok"}
```n
## 2. POST /tasks (создание)

```powershell
curl.exe -i -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d "@requests/task1.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json
Content-Length: 41
{"id":1,"title":"Buy milk","done":false}
```n
## 3. POST /tasks (вторая задача)

```powershell
curl.exe -i -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d "@requests/task2.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json
Content-Length: 45
{"id":2,"title":"Write report","done":false}
```n
## 4. GET /tasks

```powershell
curl.exe -i http://localhost:8080/tasks
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 88
[{"id":1,"title":"Buy milk","done":false},{"id":2,"title":"Write report","done":false}]
```n
## 5. GET /tasks?q=milk

```powershell
curl.exe -i "http://localhost:8080/tasks?q=milk"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 43
[{"id":1,"title":"Buy milk","done":false}]
```n
## 6. GET /tasks/1

```powershell
curl.exe -i http://localhost:8080/tasks/1
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 41
{"id":1,"title":"Buy milk","done":false}
```n
## 7. PATCH /tasks/1

```powershell
curl.exe -i -X PATCH http://localhost:8080/tasks/1 -H "Content-Type: application/json" -d "@requests/done.json"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 40
{"id":1,"title":"Buy milk","done":true}
```n
## 8. POST без title (400)

```powershell
curl.exe -i -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d "@requests/empty.json"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Content-Length: 30
{"error":"title is required"}
```n
## 9. POST слишком короткий title (422)

```powershell
curl.exe -i -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d "@requests/short.json"
```n
Ответ:

```text
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/json
Content-Length: 62
{"error":"title length must be between 3 and 140 characters"}
```n
## 10. GET /tasks/abc (400)

```powershell
curl.exe -i http://localhost:8080/tasks/abc
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Content-Length: 23
{"error":"invalid id"}
```n
## 11. GET /tasks/9999 (404)

```powershell
curl.exe -i http://localhost:8080/tasks/9999
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json
Content-Length: 27
{"error":"task not found"}
```n
## 12. DELETE /tasks/2 (204)

```powershell
curl.exe -i -X DELETE http://localhost:8080/tasks/2
```n
Ответ:

```text
HTTP/1.1 204 No Content
```n
