package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rudolf31/pz4-todo/internal/task"
)

func newTestRouter(t *testing.T, path string) http.Handler {
	t.Helper()
	repo, err := task.NewRepo(path)
	if err != nil {
		t.Fatal(err)
	}
	return newRouter(repo)
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeTasks(t *testing.T, rec *httptest.ResponseRecorder) []task.Task {
	t.Helper()
	var out []task.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return out
}

func TestHealth(t *testing.T) {
	rec := do(newTestRouter(t, ""), "GET", "/health", "")
	if rec.Code != 200 || rec.Body.String() != "OK" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestCRUD(t *testing.T) {
	h := newTestRouter(t, "")

	rec := do(h, "POST", "/api/v1/tasks", `{"title":"Learn chi"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	rec = do(h, "GET", "/api/v1/tasks/1", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"title":"Learn chi"`) {
		t.Fatalf("get = %d %s", rec.Code, rec.Body.String())
	}

	rec = do(h, "PUT", "/api/v1/tasks/1", `{"title":"Learn chi deeper","done":true}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Fatalf("put = %d %s", rec.Code, rec.Body.String())
	}

	if rec = do(h, "DELETE", "/api/v1/tasks/1", ""); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete = %d body=%q", rec.Code, rec.Body.String())
	}
	if rec = do(h, "GET", "/api/v1/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete = %d", rec.Code)
	}
}

func TestErrors(t *testing.T) {
	h := newTestRouter(t, "")
	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"bad json", "POST", "/api/v1/tasks", `{`, 400},
		{"empty title", "POST", "/api/v1/tasks", `{"title":"  "}`, 400},
		{"short title", "POST", "/api/v1/tasks", `{"title":"ab"}`, 422},
		{"long title", "POST", "/api/v1/tasks", `{"title":"` + strings.Repeat("x", 101) + `"}`, 422},
		{"bad id", "GET", "/api/v1/tasks/abc", "", 400},
		{"zero id", "GET", "/api/v1/tasks/0", "", 400},
		{"not found get", "GET", "/api/v1/tasks/99", "", 404},
		{"not found put", "PUT", "/api/v1/tasks/99", `{"title":"Valid title"}`, 404},
		{"not found delete", "DELETE", "/api/v1/tasks/99", "", 404},
		{"put bad title", "PUT", "/api/v1/tasks/1", `{"title":""}`, 400},
		{"bad done", "GET", "/api/v1/tasks?done=maybe", "", 400},
		{"bad page", "GET", "/api/v1/tasks?page=0", "", 400},
		{"bad limit", "GET", "/api/v1/tasks?limit=1000", "", 400},
		{"method not allowed", "PATCH", "/api/v1/tasks/1", `{}`, 405},
		{"unknown route", "GET", "/api/tasks", "", 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if rec := do(h, c.method, c.path, c.body); rec.Code != c.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestWrongContentType(t *testing.T) {
	h := newTestRouter(t, "")
	req := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader(`title=x`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want 415", rec.Code)
	}
}

func TestFilterAndPagination(t *testing.T) {
	h := newTestRouter(t, "")
	for _, title := range []string{"Task one", "Task two", "Task three", "Task four", "Task five"} {
		do(h, "POST", "/api/v1/tasks", `{"title":"`+title+`"}`)
	}
	do(h, "PUT", "/api/v1/tasks/2", `{"title":"Task two","done":true}`)
	do(h, "PUT", "/api/v1/tasks/4", `{"title":"Task four","done":true}`)

	if got := decodeTasks(t, do(h, "GET", "/api/v1/tasks", "")); len(got) != 5 {
		t.Errorf("all = %d, want 5", len(got))
	}

	rec := do(h, "GET", "/api/v1/tasks?done=true", "")
	if got := decodeTasks(t, rec); len(got) != 2 || got[0].ID != 2 || got[1].ID != 4 {
		t.Errorf("done=true = %+v", got)
	}

	rec = do(h, "GET", "/api/v1/tasks?page=2&limit=2", "")
	got := decodeTasks(t, rec)
	if len(got) != 2 || got[0].ID != 3 || got[1].ID != 4 {
		t.Errorf("page 2 = %+v", got)
	}
	if rec.Header().Get("X-Total-Count") != "5" {
		t.Errorf("X-Total-Count = %q", rec.Header().Get("X-Total-Count"))
	}

	if got := decodeTasks(t, do(h, "GET", "/api/v1/tasks?page=9&limit=2", "")); len(got) != 0 {
		t.Errorf("page 9 = %+v, want empty", got)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	h := newTestRouter(t, path)
	do(h, "POST", "/api/v1/tasks", `{"title":"Survive restart"}`)
	do(h, "POST", "/api/v1/tasks", `{"title":"Delete me"}`)
	do(h, "DELETE", "/api/v1/tasks/2", "")

	// «Перезапуск»: новое хранилище читает тот же файл.
	h = newTestRouter(t, path)
	got := decodeTasks(t, do(h, "GET", "/api/v1/tasks", ""))
	if len(got) != 1 || got[0].Title != "Survive restart" {
		t.Fatalf("after restart = %+v", got)
	}
	// Счётчик ID продолжается с максимального сохранённого.
	rec := do(h, "POST", "/api/v1/tasks", `{"title":"Next one"}`)
	if !strings.Contains(rec.Body.String(), `"id":2`) {
		t.Errorf("next id: %s", rec.Body.String())
	}
}

func TestCORS(t *testing.T) {
	h := newTestRouter(t, "")
	rec := do(h, "OPTIONS", "/api/v1/tasks", "")
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("preflight = %d %v", rec.Code, rec.Header())
	}
}
