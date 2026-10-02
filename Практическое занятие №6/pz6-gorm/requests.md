# Тестовые запросы (curl)

Сервер: `go run ./cmd/server` (нужна переменная `DB_DSN`, см. README). Команды для Windows PowerShell; в bash вместо `curl.exe` пишите `curl`. Тела POST лежат в папке `requests/`. Выполняйте по порядку на пустой БД (id в запросах рассчитаны на это).

| № | Запрос | Ожидаемый код | Фактический код |
|---|--------|---------------|-----------------|
| 1 | GET /health | 200 | 200 |
| 2 | POST /users | 201 | 201 |
| 3 | POST /users | 409 | 409 |
| 4 | POST /users | 400 | 400 |
| 5 | POST /notes | 201 | 201 |
| 6 | POST /notes | 201 | 201 |
| 7 | GET /notes/1 | 200 | 200 |
| 8 | GET /users/1 | 200 | 200 |
| 9 | POST /notes | 422 | 422 |
| 10 | GET /notes/99 | 404 | 404 |
| 11 | GET /notes/abc | 400 | 400 |

## 1. Проверка здоровья

```powershell
curl.exe -i http://localhost:8080/health
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 16
{"status":"ok"}
```n
## 2. Создание пользователя

```powershell
curl.exe -i -X POST http://localhost:8080/users -H "Content-Type: application/json" -d "@requests/user1.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Content-Length: 148
{"id":1,"name":"Alice","email":"alice@example.com","createdAt":"2026-10-02T12:41:55.7444464+03:00","updatedAt":"2026-10-02T12:41:55.7444464+03:00"}
```n
## 3. Дубликат email (uniqueIndex)

```powershell
curl.exe -i -X POST http://localhost:8080/users -H "Content-Type: application/json" -d "@requests/user_dup.json"
```n
Ответ:

```text
HTTP/1.1 409 Conflict
Content-Type: application/json; charset=utf-8
Content-Length: 48
{"error":"user with this email already exists"}
```n
## 4. Пользователь без email

```powershell
curl.exe -i -X POST http://localhost:8080/users -H "Content-Type: application/json" -d "@requests/user_bad.json"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
Content-Length: 40
{"error":"name and email are required"}
```n
## 5. Создание заметки с тегами go, gorm

```powershell
curl.exe -i -X POST http://localhost:8080/notes -H "Content-Type: application/json" -d "@requests/note1.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Content-Length: 572
{"id":1,"title":"Первая заметка","content":"Текст...","userId":1,"user":{"id":1,"name":"Alice","email":"alice@example.com","createdAt":"2026-10-02T12:41:55.744446+03:00","updatedAt":"2026-10-02T12:41:55.744446+03:00"},"tags":[{"id":1,"name":"go","createdAt":"2026-10-02T12:41:56.002553+03:00","updatedAt":"2026-10-02T12:41:56.002553+03:00"},{"id":2,"name":"gorm","createdAt":"2026-10-02T12:41:56.010814+03:00","updatedAt":"2026-10-02T12:41:56.010814+03:00"}],"createdAt":"2026-10-02T12:41:56.014455+03:00","updatedAt":"2026-10-02T12:41:56.014455+03:00"}
```n
## 6. Вторая заметка (тег go переиспользован)

```powershell
curl.exe -i -X POST http://localhost:8080/notes -H "Content-Type: application/json" -d "@requests/note2.json"
```n
Ответ:

```text
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Content-Length: 566
{"id":2,"title":"Про SQL","content":"JOIN-ы и индексы","userId":1,"user":{"id":1,"name":"Alice","email":"alice@example.com","createdAt":"2026-10-02T12:41:55.744446+03:00","updatedAt":"2026-10-02T12:41:55.744446+03:00"},"tags":[{"id":1,"name":"go","createdAt":"2026-10-02T12:41:56.002553+03:00","updatedAt":"2026-10-02T12:41:56.002553+03:00"},{"id":3,"name":"sql","createdAt":"2026-10-02T12:41:56.127752+03:00","updatedAt":"2026-10-02T12:41:56.127752+03:00"}],"createdAt":"2026-10-02T12:41:56.130497+03:00","updatedAt":"2026-10-02T12:41:56.130497+03:00"}
```n
## 7. Заметка с автором и тегами (Preload)

```powershell
curl.exe -i http://localhost:8080/notes/1
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 572
{"id":1,"title":"Первая заметка","content":"Текст...","userId":1,"user":{"id":1,"name":"Alice","email":"alice@example.com","createdAt":"2026-10-02T12:41:55.744446+03:00","updatedAt":"2026-10-02T12:41:55.744446+03:00"},"tags":[{"id":1,"name":"go","createdAt":"2026-10-02T12:41:56.002553+03:00","updatedAt":"2026-10-02T12:41:56.002553+03:00"},{"id":2,"name":"gorm","createdAt":"2026-10-02T12:41:56.010814+03:00","updatedAt":"2026-10-02T12:41:56.010814+03:00"}],"createdAt":"2026-10-02T12:41:56.014455+03:00","updatedAt":"2026-10-02T12:41:56.014455+03:00"}
```n
## 8. Пользователь с заметками и тегами (Preload Notes.Tags)

```powershell
curl.exe -i http://localhost:8080/users/1
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 988
{"id":1,"name":"Alice","email":"alice@example.com","notes":[{"id":1,"title":"Первая заметка","content":"Текст...","userId":1,"tags":[{"id":1,"name":"go","createdAt":"2026-10-02T12:41:56.002553+03:00","updatedAt":"2026-10-02T12:41:56.002553+03:00"},{"id":2,"name":"gorm","createdAt":"2026-10-02T12:41:56.010814+03:00","updatedAt":"2026-10-02T12:41:56.010814+03:00"}],"createdAt":"2026-10-02T12:41:56.014455+03:00","updatedAt":"2026-10-02T12:41:56.014455+03:00"},{"id":2,"title":"Про SQL","content":"JOIN-ы и индексы","userId":1,"tags":[{"id":1,"name":"go","createdAt":"2026-10-02T12:41:56.002553+03:00","updatedAt":"2026-10-02T12:41:56.002553+03:00"},{"id":3,"name":"sql","createdAt":"2026-10-02T12:41:56.127752+03:00","updatedAt":"2026-10-02T12:41:56.127752+03:00"}],"createdAt":"2026-10-02T12:41:56.130497+03:00","updatedAt":"2026-10-02T12:41:56.130497+03:00"}],"createdAt":"2026-10-02T12:41:55.744446+03:00","updatedAt":"2026-10-02T12:41:55.744446+03:00"}
```n
## 9. Заметка несуществующего пользователя

```powershell
curl.exe -i -X POST http://localhost:8080/notes -H "Content-Type: application/json" -d "@requests/note_nouser.json"
```n
Ответ:

```text
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/json; charset=utf-8
Content-Length: 27
{"error":"user not found"}
```n
## 10. Несуществующая заметка

```powershell
curl.exe -i http://localhost:8080/notes/99
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
Content-Length: 27
{"error":"note not found"}
```n
## 11. Некорректный id

```powershell
curl.exe -i http://localhost:8080/notes/abc
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
Content-Length: 19
{"error":"bad id"}
```n
