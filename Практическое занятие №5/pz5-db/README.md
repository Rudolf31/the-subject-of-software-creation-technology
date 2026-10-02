# pz5-db

Подключение к PostgreSQL через `database/sql` (драйвер `pgx/v5/stdlib`): параметризованные `INSERT` и `SELECT`, `context` с таймаутами, пул соединений, транзакции (практическое занятие №5).

## Окружение

- Go 1.25 (версия зафиксирована в go.mod из-за pgx; при установленном Go 1.24 нужный toolchain скачивается автоматически), Windows 11
- PostgreSQL 18 в Docker-контейнере (нативный Postgres не нужен)

```powershell
docker run -d --name pz5-postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:latest
docker exec pz5-postgres psql -U postgres -c "CREATE DATABASE todo;"
Get-Content schema.sql -Raw | docker exec -i pz5-postgres psql -U postgres -d todo
```

Если Postgres установлен локально — те же команды через `psql -U postgres`.

## Настройка подключения

DSN берётся из переменной `DATABASE_URL` (можно через файл `.env`, пример — `.env.example`). Если переменная не задана, используется учебный DSN для контейнера выше:

```
postgres://postgres:postgres@127.0.0.1:5432/todo?sslmode=disable
```

Используется `127.0.0.1`, а не `localhost`: на Windows `localhost` сначала резолвится в IPv6 (`::1`), и каждое новое подключение к Docker-контейнеру задерживается примерно на секунду.

## Запуск

```powershell
go run .            # вставка задач, транзакция, выборки
go run . -bench     # замеры настроек пула
go test ./...       # интеграционные тесты (нужен запущенный Postgres)
```

Интеграционные тесты работают в отдельной временной схеме и удаляют её после себя; если БД недоступна — пропускаются.

## Что реализовано

- `CreateTask` — `INSERT ... RETURNING id`
- `ListTasks` — `SELECT` всех задач
- `ListDone(ctx, done)` — `WHERE done = $1`
- `FindByID(ctx, id)` — `QueryRow` + `ErrNotFound`
- `CreateMany(ctx, titles)` — массовая вставка в одной транзакции (откат при ошибке)
- `MarkDone(ctx, id)` — `UPDATE`, нужен для демонстрации `ListDone(true)`

Везде плейсхолдеры `$1`, `$2`, ... и `context` с таймаутом; конкатенации SQL нет.

## Пул соединений

`SetMaxOpenConns(10)`, `SetMaxIdleConns(5)`, `SetConnMaxLifetime(30m)`. Обоснование и замеры — в отчёте; повторить: `go run . -bench`.

## Структура

```
pz5-db/
├─ main.go             # сценарий: INSERT, транзакция, SELECT, фильтр, поиск
├─ db.go               # openDB: пул, Ping с таймаутом
├─ repository.go       # Repo и SQL-запросы
├─ bench.go            # замеры пула (-bench)
├─ repository_test.go  # интеграционные тесты
├─ schema.sql          # таблица tasks
├─ .env.example
└─ go.mod
```
