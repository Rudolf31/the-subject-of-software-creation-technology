# Тестовые запросы (curl)

Сервер: `go run ./cmd/server` (Redis на 127.0.0.1:6379, для `/tasks/{id}` ещё PostgreSQL с БД `todo` из ПЗ №5). Команды для Windows PowerShell; в bash вместо `curl.exe` пишите `curl`. Сначала выполняйте шаги по порядку.

| № | Сценарий | Ожидаемый код | Фактический код |
|---|----------|---------------|-----------------|
| 1 | Проверка здоровья | 200 | 200 |
| 2 | Сохранить test=hello (TTL 10 с) | 200 | 200 |
| 3 | Прочитать test | 200 | 200 |
| 4 | Узнать TTL | 200 | 200 |
| 5 | Свой TTL: session=abc, 60 секунд | 200 | 200 |
| 6 | TTL ключа session | 200 | 200 |
| 7 | Прочитать test через 11 с (TTL истёк) | 404 | 404 |
| 8 | TTL истёкшего ключа | 404 | 404 |
| 9 | Удалить session | 200 | 200 |
| 10 | Удалить session повторно | 404 | 404 |
| 11 | Нет value (валидация) | 400 | 400 |
| 12 | Некорректный ttl | 400 | 400 |
| 13 | Кэш задачи: первый запрос (MISS, из PostgreSQL) | 200 | 200 |
| 14 | Кэш задачи: второй запрос (HIT, из Redis) | 200 | 200 |
| 15 | Несуществующая задача | 404 | 404 |

## 1. Проверка здоровья

```powershell
curl.exe -i http://localhost:8080/health
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
OK
```n
## 2. Сохранить test=hello (TTL 10 с)

```powershell
curl.exe -i "http://localhost:8080/set?key=test&value=hello"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
OK: test=hello (TTL 10s)
```n
## 3. Прочитать test

```powershell
curl.exe -i "http://localhost:8080/get?key=test"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
VALUE: test=hello
```n
## 4. Узнать TTL

```powershell
curl.exe -i "http://localhost:8080/ttl?key=test"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
TTL for test: 10s
```n
## 5. Свой TTL: session=abc, 60 секунд

```powershell
curl.exe -i "http://localhost:8080/set?key=session&value=abc&ttl=60"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
OK: session=abc (TTL 1m0s)
```n
## 6. TTL ключа session

```powershell
curl.exe -i "http://localhost:8080/ttl?key=session"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
TTL for session: 1m0s
```n
## 7. Прочитать test через 11 с (TTL истёк)

```powershell
curl.exe -i "http://localhost:8080/get?key=test"
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
key not found (missing or expired)
```n
## 8. TTL истёкшего ключа

```powershell
curl.exe -i "http://localhost:8080/ttl?key=test"
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
key not found (missing or expired)
```n
## 9. Удалить session

```powershell
curl.exe -i "http://localhost:8080/del?key=session"
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
DELETED: session
```n
## 10. Удалить session повторно

```powershell
curl.exe -i "http://localhost:8080/del?key=session"
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
key not found
```n
## 11. Нет value (валидация)

```powershell
curl.exe -i "http://localhost:8080/set?key=a"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
key and value required
```n
## 12. Некорректный ttl

```powershell
curl.exe -i "http://localhost:8080/set?key=a&value=b&ttl=abc"
```n
Ответ:

```text
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
ttl must be an integer between 1 and 3600 seconds
```n
## 13. Кэш задачи: первый запрос (MISS, из PostgreSQL)

```powershell
curl.exe -i http://localhost:8080/tasks/1
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Cache: MISS
{"id":1,"title":"Первая задача из psql","done":false,"created_at":"2026-10-02T11:08:14.58401+03:00"}
```n
## 14. Кэш задачи: второй запрос (HIT, из Redis)

```powershell
curl.exe -i http://localhost:8080/tasks/1
```n
Ответ:

```text
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Cache: HIT
{"id":1,"title":"Первая задача из psql","done":false,"created_at":"2026-10-02T11:08:14.58401+03:00"}
```n
## 15. Несуществующая задача

```powershell
curl.exe -i http://localhost:8080/tasks/999
```n
Ответ:

```text
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
task not found
```n
