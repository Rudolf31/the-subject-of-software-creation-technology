# myapp

Учебный Go-проект (практическое занятие №2): минимальный HTTP-сервис со структурой каталогов по мотивам [project-layout](https://github.com/golang-standards/project-layout). Есть маршруты `/` (текст), `/ping` (JSON), `/fail` (пример JSON-ошибки), логирование запросов и заголовок `X-Request-Id`.

## Запуск

```powershell
go run ./cmd/myapp
```

Сервер слушает `:8080`.

## Сборка

```powershell
go build -o bin\myapp.exe ./cmd/myapp
.\bin\myapp.exe
```

## Проверка

```powershell
curl -i http://localhost:8080/
curl -i http://localhost:8080/ping
curl -i -H "X-Request-Id: demo-123" http://localhost:8080/ping
curl -i http://localhost:8080/fail
```

## Структура

```
myapp/
├─ cmd/myapp/main.go              # точка входа
├─ internal/app/app.go            # сборка сервера, middleware
├─ internal/app/handlers/ping.go  # обработчик /ping
├─ utils/logger.go                # логирование, генерация request-id
├─ utils/httpjson.go              # JSON-ответы и ошибки
├─ go.mod
└─ README.md
```
