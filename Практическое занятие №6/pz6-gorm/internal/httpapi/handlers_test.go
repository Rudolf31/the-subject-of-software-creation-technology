package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/Rudolf31/pz6-gorm/internal/db"
	"github.com/Rudolf31/pz6-gorm/internal/models"
)

const defaultDSN = "host=127.0.0.1 user=postgres password=postgres dbname=pz6_gorm port=5432 sslmode=disable"

// newTestServer поднимает роутер на отдельной временной схеме PostgreSQL;
// схема удаляется после теста. Если БД недоступна — тест пропускается.
func newTestServer(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}

	admin, err := db.Connect(dsn)
	if err != nil {
		t.Skipf("PostgreSQL недоступен: %v", err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Skipf("PostgreSQL недоступен: %v", err)
	}
	sqlAdmin, _ := admin.DB()
	t.Cleanup(func() {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		sqlAdmin.Close()
	})

	d, err := db.Connect(dsn + " search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	sqlD, _ := d.DB()
	t.Cleanup(func() { sqlD.Close() })

	if err := d.AutoMigrate(&models.User{}, &models.Note{}, &models.Tag{}); err != nil {
		t.Fatal(err)
	}
	return BuildRouter(d), d
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

func mustUser(t *testing.T, h http.Handler, name, email string) {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"email":%q}`, name, email)
	if rec := do(h, "POST", "/users", body); rec.Code != http.StatusCreated {
		t.Fatalf("create user = %d %s", rec.Code, rec.Body.String())
	}
}

func TestHealth(t *testing.T) {
	h, _ := newTestServer(t)
	if rec := do(h, "GET", "/health", ""); rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateUserAndDuplicateEmail(t *testing.T) {
	h, _ := newTestServer(t)

	rec := do(h, "POST", "/users", `{"name":"Alice","email":"alice@example.com"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"id":1`) {
		t.Fatalf("create = %d %s", rec.Code, rec.Body.String())
	}
	// повтор с тем же email нарушает uniqueIndex → 409
	if rec := do(h, "POST", "/users", `{"name":"Other","email":"alice@example.com"}`); rec.Code != http.StatusConflict {
		t.Errorf("duplicate = %d, want 409 (%s)", rec.Code, rec.Body.String())
	}
	for _, body := range []string{`{`, `{"name":"x"}`, `{"name":" ","email":"a@b"}`} {
		if rec := do(h, "POST", "/users", body); rec.Code != http.StatusBadRequest {
			t.Errorf("body %s = %d, want 400", body, rec.Code)
		}
	}
}

func TestCreateNoteWithTagsAndPreload(t *testing.T) {
	h, _ := newTestServer(t)
	mustUser(t, h, "Alice", "alice@example.com")

	rec := do(h, "POST", "/notes", `{"title":"First","content":"text","userId":1,"tags":["go","gorm"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create note = %d %s", rec.Code, rec.Body.String())
	}
	var note models.Note
	if err := json.Unmarshal(rec.Body.Bytes(), &note); err != nil {
		t.Fatal(err)
	}
	if note.User == nil || note.User.Email != "alice@example.com" || len(note.Tags) != 2 {
		t.Errorf("preload missing: %+v", note)
	}

	// второй заметке достаётся уже существующий тег «go» (FirstOrCreate), без дублей
	do(h, "POST", "/notes", `{"title":"Second","userId":1,"tags":["go","  go ","","sql"]}`)

	rec = do(h, "GET", "/notes/2", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get note = %d", rec.Code)
	}
	json.Unmarshal(rec.Body.Bytes(), &note)
	if len(note.Tags) != 2 {
		t.Errorf("tags = %+v, want go+sql", note.Tags)
	}

	// у пользователя две заметки с тегами (Preload("Notes.Tags"))
	rec = do(h, "GET", "/users/1", "")
	var user models.User
	json.Unmarshal(rec.Body.Bytes(), &user)
	if rec.Code != 200 || len(user.Notes) != 2 || len(user.Notes[0].Tags) != 2 {
		t.Errorf("user = %d %s", rec.Code, rec.Body.String())
	}
}

func TestTagsAreShared(t *testing.T) {
	h, d := newTestServer(t)
	mustUser(t, h, "Alice", "alice@example.com")
	do(h, "POST", "/notes", `{"title":"A","userId":1,"tags":["go"]}`)
	do(h, "POST", "/notes", `{"title":"B","userId":1,"tags":["go"]}`)

	var tags int64
	d.Model(&models.Tag{}).Count(&tags)
	var links int64
	d.Table("note_tags").Count(&links)
	if tags != 1 || links != 2 {
		t.Errorf("tags = %d, links = %d; want 1 and 2", tags, links)
	}
}

func TestNoteErrors(t *testing.T) {
	h, _ := newTestServer(t)
	mustUser(t, h, "Alice", "alice@example.com")

	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"bad json", "POST", "/notes", `{`, 400},
		{"no title", "POST", "/notes", `{"userId":1}`, 400},
		{"no user id", "POST", "/notes", `{"title":"x"}`, 400},
		{"unknown user", "POST", "/notes", `{"title":"x","userId":99}`, 422},
		{"bad id", "GET", "/notes/abc", "", 400},
		{"note not found", "GET", "/notes/99", "", 404},
		{"user not found", "GET", "/users/99", "", 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if rec := do(h, c.method, c.path, c.body); rec.Code != c.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestFailedNoteLeavesNoTags(t *testing.T) {
	h, d := newTestServer(t)
	// пользователя нет → транзакция откатывается, теги не создаются
	do(h, "POST", "/notes", `{"title":"x","userId":99,"tags":["orphan"]}`)

	var tags int64
	d.Model(&models.Tag{}).Count(&tags)
	if tags != 0 {
		t.Errorf("tags = %d, want 0", tags)
	}
}
