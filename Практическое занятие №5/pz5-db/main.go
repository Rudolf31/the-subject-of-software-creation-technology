package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Учебный DSN для локального контейнера из README; в реальных проектах — только из окружения.
const fallbackDSN = "postgres://postgres:postgres@127.0.0.1:5432/todo?sslmode=disable"

func main() {
	benchMode := flag.Bool("bench", false, "замерить настройки пула соединений и выйти")
	flag.Parse()

	// .env не обязателен; если файла нет — ошибка игнорируется
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fallbackDSN
	}

	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("openDB error: %v", err)
	}
	defer db.Close()

	if *benchMode {
		runBench(dsn)
		return
	}

	repo := NewRepo(db)

	// 1) Вставим пару задач
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	titles := []string{"Сделать ПЗ №5", "Купить кофе", "Проверить отчёты"}
	var firstID int
	for i, title := range titles {
		id, err := repo.CreateTask(ctx, title)
		if err != nil {
			log.Fatalf("CreateTask error: %v", err)
		}
		if i == 0 {
			firstID = id
		}
		log.Printf("Inserted task id=%d (%s)", id, title)
	}

	// 2) Массовая вставка в одной транзакции
	if err := repo.CreateMany(ctx, []string{"Пакетная задача 1", "Пакетная задача 2"}); err != nil {
		log.Fatalf("CreateMany error: %v", err)
	}
	log.Println("CreateMany: 2 tasks inserted in one transaction")

	// 3) Отметим первую задачу выполненной
	if err := repo.MarkDone(ctx, firstID); err != nil {
		log.Fatalf("MarkDone error: %v", err)
	}

	// 4) Прочитаем весь список задач
	tasks, err := repo.ListTasks(ctx)
	if err != nil {
		log.Fatalf("ListTasks error: %v", err)
	}
	fmt.Println("=== Tasks ===")
	printTasks(tasks)

	// 5) Фильтр по done
	for _, done := range []bool{true, false} {
		list, err := repo.ListDone(ctx, done)
		if err != nil {
			log.Fatalf("ListDone error: %v", err)
		}
		fmt.Printf("=== done=%v (%d) ===\n", done, len(list))
		printTasks(list)
	}

	// 6) Поиск по id
	t, err := repo.FindByID(ctx, firstID)
	if err != nil {
		log.Fatalf("FindByID error: %v", err)
	}
	fmt.Printf("=== FindByID(%d) ===\n#%d | %s | done=%v | created %s\n",
		firstID, t.ID, t.Title, t.Done, t.CreatedAt.Format(time.RFC3339))

	if _, err := repo.FindByID(ctx, 999999); err != nil {
		fmt.Printf("FindByID(999999): %v\n", err)
	}
}

func printTasks(tasks []Task) {
	for _, t := range tasks {
		fmt.Printf("#%d | %-24s | done=%-5v | %s\n",
			t.ID, t.Title, t.Done, t.CreatedAt.Format(time.RFC3339))
	}
}
