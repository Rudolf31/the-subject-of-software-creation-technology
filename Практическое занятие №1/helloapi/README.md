# helloapi

Минимальный HTTP-сервис на Go (`net/http`) для практического занятия №1 по дисциплине «Технологии создания программного обеспечения».

## Требования

- Go 1.22+ (проверено на 1.24.1)
- Git
- Внешняя зависимость: [`github.com/google/uuid`](https://github.com/google/uuid) (подтягивается автоматически)

## Маршруты

| Маршрут   | Ответ                                                  |
|-----------|--------------------------------------------------------|
| `/hello`  | текст `Hello, world!`                                  |
| `/user`   | JSON `{"id":"<UUID>","name":"Gopher"}`                 |
| `/health` | JSON `{"status":"ok","time":"<RFC3339>"}`              |

Каждый запрос пишется в лог (адрес, метод, путь, время обработки).

## Запуск

```powershell
go run ./cmd/server
```

## Сборка

```powershell
go build -o helloapi.exe ./cmd/server
.\helloapi.exe
```

## Примеры запросов

```powershell
curl http://localhost:8080/hello
curl http://localhost:8080/user
curl http://localhost:8080/health
```

Пример ответа `/user`:

```json
{"id":"3f1c1c5c-9a1b-4a6a-8a26-7b3f2f7d8b0b","name":"Gopher"}
```

## Конфигурация

Порт задаётся переменной окружения `APP_PORT` (по умолчанию `8080`):

```powershell
$env:APP_PORT="8081"
go run ./cmd/server
```

## Проверка кода

```powershell
go fmt ./...
go vet ./...
```

## Структура проекта

```
helloapi/
  cmd/
    server/
      main.go
  go.mod
  go.sum
  README.md
```
