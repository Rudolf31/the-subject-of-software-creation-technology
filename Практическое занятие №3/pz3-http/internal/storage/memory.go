package storage

import (
	"errors"
	"sort"
	"sync"
)

// ErrNotFound возвращается, когда задачи с таким ID нет.
var ErrNotFound = errors.New("not found")

type Task struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type MemoryStore struct {
	mu    sync.RWMutex
	auto  int64
	tasks map[int64]*Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[int64]*Task),
	}
}

// Create добавляет задачу и возвращает её копию.
func (s *MemoryStore) Create(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auto++
	t := &Task{ID: s.auto, Title: title, Done: false}
	s.tasks[t.ID] = t
	return *t
}

func (s *MemoryStore) Get(id int64) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return *t, nil
}

// List возвращает все задачи, отсортированные по ID.
func (s *MemoryStore) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Update меняет поля задачи (nil — оставить как есть).
func (s *MemoryStore) Update(id int64, title *string, done *bool) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if title != nil {
		t.Title = *title
	}
	if done != nil {
		t.Done = *done
	}
	return *t, nil
}

func (s *MemoryStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
