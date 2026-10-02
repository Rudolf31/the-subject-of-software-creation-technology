package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

//go:embed schema.sql
var schemaSQL string

// newTestRepo создаёт отдельную схему, чтобы тесты не трогали реальные данные,
// и удаляет её по завершении. Если БД недоступна — тест пропускается.
func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fallbackDSN
	}

	admin, err := openDB(dsn)
	if err != nil {
		t.Skipf("PostgreSQL недоступен: %v", err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := openDB(dsn + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	return NewRepo(db)
}

func TestCreateAndList(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id1, err := repo.CreateTask(ctx, "first")
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := repo.CreateTask(ctx, "second")
	if id2 != id1+1 {
		t.Errorf("ids = %d, %d", id1, id2)
	}

	tasks, err := repo.ListTasks(ctx)
	if err != nil || len(tasks) != 2 || tasks[0].Title != "first" || tasks[0].Done {
		t.Fatalf("list = %+v, err = %v", tasks, err)
	}
}

func TestListDoneAndFindByID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	a, _ := repo.CreateTask(ctx, "a")
	repo.CreateTask(ctx, "b")
	if err := repo.MarkDone(ctx, a); err != nil {
		t.Fatal(err)
	}

	done, _ := repo.ListDone(ctx, true)
	todo, _ := repo.ListDone(ctx, false)
	if len(done) != 1 || done[0].ID != a || len(todo) != 1 || todo[0].Title != "b" {
		t.Errorf("done = %+v, todo = %+v", done, todo)
	}

	got, err := repo.FindByID(ctx, a)
	if err != nil || got.Title != "a" || !got.Done {
		t.Errorf("FindByID = %+v, %v", got, err)
	}
	if _, err := repo.FindByID(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindByID missing: err = %v", err)
	}
	if err := repo.MarkDone(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Errorf("MarkDone missing: err = %v", err)
	}
}

func TestCreateManyCommit(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateMany(ctx, []string{"x", "y", "z"}); err != nil {
		t.Fatal(err)
	}
	if tasks, _ := repo.ListTasks(ctx); len(tasks) != 3 {
		t.Errorf("got %d tasks, want 3", len(tasks))
	}
}

func TestCreateManyRollback(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// NUL-символ недопустим в TEXT: вторая вставка падает, первая должна откатиться
	err := repo.CreateMany(ctx, []string{"ok", "bad\x00title"})
	if err == nil {
		t.Fatal("expected error")
	}
	if tasks, _ := repo.ListTasks(ctx); len(tasks) != 0 {
		t.Errorf("rollback failed: %+v", tasks)
	}
}

func TestSQLInjectionIsHarmless(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	evil := `x'); DROP TABLE tasks; --`
	if _, err := repo.CreateTask(ctx, evil); err != nil {
		t.Fatal(err)
	}
	tasks, err := repo.ListTasks(ctx)
	if err != nil || len(tasks) != 1 || tasks[0].Title != evil {
		t.Fatalf("title stored incorrectly: %+v, err = %v", tasks, err)
	}
}

func TestContextTimeout(t *testing.T) {
	repo := newTestRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := repo.DB.ExecContext(ctx, "SELECT pg_sleep(2)")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
