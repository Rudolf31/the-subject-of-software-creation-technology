package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rudolf31/pz3-http/internal/storage"
)

func newServer() http.Handler {
	return NewHandlers(storage.NewMemoryStore()).Routes()
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

func TestHealth(t *testing.T) {
	rec := do(newServer(), "GET", "/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Errorf("body = %s", got)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	h := newServer()

	rec := do(h, "POST", "/tasks", `{"title":"Buy milk"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	var created storage.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Title != "Buy milk" || created.Done {
		t.Errorf("unexpected task: %+v", created)
	}

	rec = do(h, "GET", "/tasks/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rec.Code)
	}
}

func TestCreateTaskErrors(t *testing.T) {
	h := newServer()
	cases := []struct {
		name, body string
		want       int
	}{
		{"empty title", `{"title":""}`, http.StatusBadRequest},
		{"missing title", `{}`, http.StatusBadRequest},
		{"bad json", `{`, http.StatusBadRequest},
		{"too short", `{"title":"ab"}`, http.StatusUnprocessableEntity},
		{"too long", `{"title":"` + strings.Repeat("a", 141) + `"}`, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if rec := do(h, "POST", "/tasks", c.body); rec.Code != c.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestGetTaskErrors(t *testing.T) {
	h := newServer()
	if rec := do(h, "GET", "/tasks/abc", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("abc: status = %d, want 400", rec.Code)
	}
	if rec := do(h, "GET", "/tasks/9999", ""); rec.Code != http.StatusNotFound {
		t.Errorf("9999: status = %d, want 404", rec.Code)
	}
}

func TestListTasksFilter(t *testing.T) {
	h := newServer()
	do(h, "POST", "/tasks", `{"title":"Buy milk"}`)
	do(h, "POST", "/tasks", `{"title":"Write report"}`)

	rec := do(h, "GET", "/tasks?q=MILK", "")
	var tasks []storage.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Buy milk" {
		t.Errorf("filtered = %+v", tasks)
	}

	rec = do(h, "GET", "/tasks", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &tasks)
	if len(tasks) != 2 {
		t.Errorf("all = %d tasks, want 2", len(tasks))
	}
}

func TestPatchAndDelete(t *testing.T) {
	h := newServer()
	do(h, "POST", "/tasks", `{"title":"Buy milk"}`)

	rec := do(h, "PATCH", "/tasks/1", `{"done":true}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Errorf("patch: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(h, "PATCH", "/tasks/1", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty patch: status = %d, want 400", rec.Code)
	}
	if rec := do(h, "PATCH", "/tasks/42", `{"done":true}`); rec.Code != http.StatusNotFound {
		t.Errorf("patch missing: status = %d, want 404", rec.Code)
	}

	if rec := do(h, "DELETE", "/tasks/1", ""); rec.Code != http.StatusNoContent {
		t.Errorf("delete: status = %d, want 204", rec.Code)
	}
	if rec := do(h, "DELETE", "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("delete again: status = %d, want 404", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	if rec := do(newServer(), "PUT", "/tasks", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestCORSAndLogging(t *testing.T) {
	h := Logging(CORS(newServer()))
	rec := do(h, "OPTIONS", "/tasks", "")
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", rec.Code)
	}
	rec = do(h, "GET", "/health", "")
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}
