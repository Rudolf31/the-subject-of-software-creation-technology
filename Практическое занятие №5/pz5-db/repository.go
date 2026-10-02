package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("task not found")

// Task — модель для сканирования результатов SELECT
type Task struct {
	ID        int
	Title     string
	Done      bool
	CreatedAt time.Time
}

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

// CreateTask — параметризованный INSERT с возвратом id
func (r *Repo) CreateTask(ctx context.Context, title string) (int, error) {
	var id int
	const q = `INSERT INTO tasks (title) VALUES ($1) RETURNING id;`
	err := r.DB.QueryRowContext(ctx, q, title).Scan(&id)
	return id, err
}

// ListTasks — базовый SELECT всех задач
func (r *Repo) ListTasks(ctx context.Context) ([]Task, error) {
	const q = `SELECT id, title, done, created_at FROM tasks ORDER BY id;`
	return r.queryTasks(ctx, q)
}

// ListDone возвращает только выполненные (done=true) или невыполненные (done=false) задачи.
func (r *Repo) ListDone(ctx context.Context, done bool) ([]Task, error) {
	const q = `SELECT id, title, done, created_at FROM tasks WHERE done = $1 ORDER BY id;`
	return r.queryTasks(ctx, q, done)
}

// FindByID возвращает одну задачу или ErrNotFound.
func (r *Repo) FindByID(ctx context.Context, id int) (*Task, error) {
	const q = `SELECT id, title, done, created_at FROM tasks WHERE id = $1;`
	var t Task
	err := r.DB.QueryRowContext(ctx, q, id).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkDone выставляет done=true; вернёт ErrNotFound, если задачи нет.
func (r *Repo) MarkDone(ctx context.Context, id int) error {
	const q = `UPDATE tasks SET done = TRUE WHERE id = $1;`
	res, err := r.DB.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateMany вставляет все задачи в одной транзакции: либо все, либо ни одной.
func (r *Repo) CreateMany(ctx context.Context, titles []string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // после успешного Commit возвращает sql.ErrTxDone — это нормально

	// один подготовленный запрос на всю пачку
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO tasks (title) VALUES ($1);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, title := range titles {
		if _, err := stmt.ExecContext(ctx, title); err != nil {
			return fmt.Errorf("insert %q: %w", title, err)
		}
	}
	return tx.Commit()
}

func (r *Repo) queryTasks(ctx context.Context, q string, args ...any) ([]Task, error) {
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
