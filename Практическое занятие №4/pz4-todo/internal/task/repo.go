package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("task not found")

// Repo — хранилище задач в памяти с необязательным сохранением в JSON-файл.
type Repo struct {
	mu    sync.RWMutex
	seq   int64
	items map[int64]*Task
	path  string // пустой путь — хранить только в памяти
}

// NewRepo создаёт хранилище. Если path не пустой, данные читаются из файла
// при старте (если он есть) и записываются после каждого изменения.
func NewRepo(path string) (*Repo, error) {
	r := &Repo{items: make(map[int64]*Task), path: path}
	if path == "" {
		return r, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, t := range tasks {
		r.items[t.ID] = t
		if t.ID > r.seq {
			r.seq = t.ID
		}
	}
	return r, nil
}

// save записывает все задачи в файл; вызывается под write-локом.
func (r *Repo) save() error {
	if r.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(r.sorted(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o644)
}

// sorted возвращает копии задач, отсортированные по ID (вызывать под локом).
func (r *Repo) sorted() []Task {
	out := make([]Task, 0, len(r.items))
	for _, t := range r.items {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Repo) List() []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sorted()
}

func (r *Repo) Get(id int64) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.items[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return *t, nil
}

func (r *Repo) Create(title string) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	now := time.Now()
	t := &Task{ID: r.seq, Title: title, CreatedAt: now, UpdatedAt: now}
	r.items[t.ID] = t
	if err := r.save(); err != nil {
		delete(r.items, t.ID)
		r.seq--
		return Task{}, err
	}
	return *t, nil
}

func (r *Repo) Update(id int64, title string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	prev := *t
	t.Title = title
	t.Done = done
	t.UpdatedAt = time.Now()
	if err := r.save(); err != nil {
		*t = prev
		return Task{}, err
	}
	return *t, nil
}

func (r *Repo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	if err := r.save(); err != nil {
		r.items[id] = t
		return err
	}
	return nil
}
