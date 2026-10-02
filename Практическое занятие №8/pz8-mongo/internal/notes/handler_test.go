package notes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newAPI(t *testing.T) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	r.Mount("/api/v1/notes", NewHandler(newTestRepo(t)).Routes())
	return r
}

func call(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
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

func createVia(t *testing.T, h http.Handler, title string) Note {
	t.Helper()
	rec := call(h, "POST", "/api/v1/notes", fmt.Sprintf(`{"title":%q,"content":"c"}`, title))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %q = %d %s", title, rec.Code, rec.Body.String())
	}
	var n Note
	json.Unmarshal(rec.Body.Bytes(), &n)
	return n
}

func TestAPICRUD(t *testing.T) {
	h := newAPI(t)

	n := createVia(t, h, "First note")
	if n.ID.IsZero() {
		t.Fatal("no id in response")
	}
	base := "/api/v1/notes/" + n.ID.Hex()

	if rec := call(h, "GET", base, ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"title":"First note"`) {
		t.Fatalf("get = %d %s", rec.Code, rec.Body.String())
	}
	rec := call(h, "PATCH", base, `{"content":"Updated content"}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"content":"Updated content"`) ||
		!strings.Contains(rec.Body.String(), `"title":"First note"`) {
		t.Fatalf("patch = %d %s", rec.Code, rec.Body.String())
	}
	if rec = call(h, "DELETE", base, ""); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete = %d %q", rec.Code, rec.Body.String())
	}
	if rec = call(h, "GET", base, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete = %d", rec.Code)
	}
}

func TestAPIErrors(t *testing.T) {
	h := newAPI(t)
	createVia(t, h, "dup")
	missing := "507f1f77bcf86cd799439011" // валидный ObjectID, которого нет

	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"bad json", "POST", "/api/v1/notes", `{`, 400},
		{"no title", "POST", "/api/v1/notes", `{"content":"x"}`, 400},
		{"blank title", "POST", "/api/v1/notes", `{"title":"   "}`, 400},
		{"long title", "POST", "/api/v1/notes", `{"title":"` + strings.Repeat("a", 201) + `"}`, 400},
		{"bad ttl", "POST", "/api/v1/notes", `{"title":"t","ttlSeconds":-5}`, 400},
		{"duplicate", "POST", "/api/v1/notes", `{"title":"dup"}`, 409},
		{"get bad id", "GET", "/api/v1/notes/abc", "", 404},
		{"get missing", "GET", "/api/v1/notes/" + missing, "", 404},
		{"patch bad json", "PATCH", "/api/v1/notes/" + missing, `{`, 400},
		{"patch empty", "PATCH", "/api/v1/notes/" + missing, `{}`, 400},
		{"patch blank title", "PATCH", "/api/v1/notes/" + missing, `{"title":""}`, 400},
		{"patch missing", "PATCH", "/api/v1/notes/" + missing, `{"content":"x"}`, 404},
		{"delete bad id", "DELETE", "/api/v1/notes/zzz", "", 404},
		{"delete missing", "DELETE", "/api/v1/notes/" + missing, "", 404},
		{"limit zero", "GET", "/api/v1/notes?limit=0", "", 400},
		{"limit too big", "GET", "/api/v1/notes?limit=1000", "", 400},
		{"skip negative", "GET", "/api/v1/notes?skip=-1", "", 400},
		{"after invalid", "GET", "/api/v1/notes?after=xyz", "", 400},
		{"skip and after", "GET", "/api/v1/notes?skip=1&after=" + missing, "", 400},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if rec := call(h, c.method, c.path, c.body); rec.Code != c.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestAPIPatchDuplicateTitle(t *testing.T) {
	h := newAPI(t)
	createVia(t, h, "one")
	two := createVia(t, h, "two")
	if rec := call(h, "PATCH", "/api/v1/notes/"+two.ID.Hex(), `{"title":"one"}`); rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestAPIListAndCursor(t *testing.T) {
	h := newAPI(t)
	for _, title := range []string{"alpha first", "beta", "gamma first", "delta", "epsilon"} {
		createVia(t, h, title)
	}

	var list []Note
	rec := call(h, "GET", "/api/v1/notes?q=first", "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	if rec.Code != 200 || len(list) != 2 {
		t.Fatalf("q=first = %d, %d items", rec.Code, len(list))
	}

	// первая страница + курсор для следующей
	rec = call(h, "GET", "/api/v1/notes?limit=2", "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	next := rec.Header().Get("X-Next-After")
	if len(list) != 2 || list[0].Title != "epsilon" || next != list[1].ID.Hex() {
		t.Fatalf("page1: %d items, X-Next-After=%q", len(list), next)
	}
	rec = call(h, "GET", "/api/v1/notes?limit=2&after="+next, "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	// порядок «от новых к старым»: epsilon, delta | gamma first, beta | alpha first
	if len(list) != 2 || list[0].Title != "gamma first" || list[1].Title != "beta" {
		t.Fatalf("page2: %+v", list)
	}
	// последняя неполная страница — без курсора
	next = rec.Header().Get("X-Next-After")
	rec = call(h, "GET", "/api/v1/notes?limit=2&after="+next, "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || rec.Header().Get("X-Next-After") != "" {
		t.Errorf("page3: %d items, cursor %q", len(list), rec.Header().Get("X-Next-After"))
	}

	// пустой результат — [] а не null
	rec = call(h, "GET", "/api/v1/notes?q=zzzz", "")
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("empty list body = %q", rec.Body.String())
	}
}

func TestAPIStats(t *testing.T) {
	h := newAPI(t)
	createVia(t, h, "a")
	rec := call(h, "GET", "/api/v1/notes/stats", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"count":1`) {
		t.Errorf("stats = %d %s", rec.Code, rec.Body.String())
	}
}

func TestAPICreateWithTTL(t *testing.T) {
	h := newAPI(t)
	rec := call(h, "POST", "/api/v1/notes", `{"title":"temp","content":"x","ttlSeconds":60}`)
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), `"expiresAt"`) {
		t.Errorf("create with ttl = %d %s", rec.Code, rec.Body.String())
	}
}
