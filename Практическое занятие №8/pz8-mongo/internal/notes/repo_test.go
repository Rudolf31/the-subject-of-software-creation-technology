package notes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Rudolf31/pz8-mongo/internal/db"
)

const defaultURI = "mongodb://root:secret@127.0.0.1:27017/?authSource=admin"

// newTestRepo подключается к MongoDB, создаёт отдельную БД pz8_test_<время> и
// удаляет её после теста. Если сервер недоступен — тест пропускается.
func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = defaultURI
	}
	ctx := context.Background()
	name := fmt.Sprintf("pz8_test_%d", time.Now().UnixNano())

	deps, err := db.ConnectMongo(ctx, uri, name)
	if err != nil {
		t.Skipf("MongoDB недоступна: %v", err)
	}
	t.Cleanup(func() {
		deps.Database.Drop(ctx)
		deps.Client.Disconnect(ctx)
	})
	r, err := NewRepo(deps.Database)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCreateAndGet(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()

	created, err := r.Create(ctx, "T1", "C1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID.IsZero() || created.ExpiresAt != nil {
		t.Fatalf("unexpected note: %+v", created)
	}
	got, err := r.ByID(ctx, created.ID.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "T1" || got.Content != "C1" || !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

func TestInvalidAndMissingID(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	for _, id := range []string{"abc", "", "123", primitive.NewObjectID().Hex()} {
		if _, err := r.ByID(ctx, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("ByID(%q) err = %v, want ErrNotFound", id, err)
		}
		if err := r.Delete(ctx, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Delete(%q) err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestUniqueTitle(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	if _, err := r.Create(ctx, "same", "a", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(ctx, "same", "b", 0); !errors.Is(err, ErrDuplicate) {
		t.Errorf("err = %v, want ErrDuplicate", err)
	}
}

func seed(t *testing.T, r *Repo, titles ...string) []Note {
	t.Helper()
	var out []Note
	for _, title := range titles {
		n, err := r.Create(context.Background(), title, "content of "+title, 0)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
		time.Sleep(2 * time.Millisecond) // разные _id по порядку
	}
	return out
}

func titles(ns []Note) []string {
	var s []string
	for _, n := range ns {
		s = append(s, n.Title)
	}
	return s
}

func TestListFilterAndSort(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	seed(t, r, "First note", "Second NOTE", "Shopping.list", "Other")

	got, err := r.List(ctx, ListParams{Limit: 10})
	if err != nil || len(got) != 4 || got[0].Title != "Other" || got[3].Title != "First note" {
		t.Fatalf("order (newest first): %v, %v", titles(got), err)
	}

	got, _ = r.List(ctx, ListParams{Q: "note", Limit: 10}) // без учёта регистра
	if len(got) != 2 {
		t.Errorf("q=note: %v", titles(got))
	}
	// спецсимволы ищутся как текст, а не как regexp
	got, _ = r.List(ctx, ListParams{Q: ".*", Limit: 10})
	if len(got) != 0 {
		t.Errorf("q=.*: %v, want none", titles(got))
	}
	got, _ = r.List(ctx, ListParams{Q: "shopping.list", Limit: 10})
	if len(got) != 1 {
		t.Errorf("q=shopping.list: %v", titles(got))
	}
}

func TestTextSearch(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	r.Create(ctx, "Go basics", "goroutines and channels", 0)
	r.Create(ctx, "Cooking", "pasta recipe", 0)

	got, err := r.List(ctx, ListParams{Search: "channels", Limit: 10})
	if err != nil || len(got) != 1 || got[0].Title != "Go basics" {
		t.Errorf("search channels: %v, %v", titles(got), err)
	}
	if got, _ = r.List(ctx, ListParams{Search: "nothing", Limit: 10}); len(got) != 0 {
		t.Errorf("search nothing: %v", titles(got))
	}
}

func TestPaginationSkipAndCursor(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	seed(t, r, "n1", "n2", "n3", "n4", "n5")

	page2, _ := r.List(ctx, ListParams{Limit: 2, Skip: 2})
	if fmt.Sprint(titles(page2)) != "[n3 n2]" {
		t.Errorf("skip page = %v", titles(page2))
	}

	// курсор: после последней заметки первой страницы
	page1, _ := r.List(ctx, ListParams{Limit: 2})
	next, _ := r.List(ctx, ListParams{Limit: 2, After: page1[len(page1)-1].ID})
	if fmt.Sprint(titles(page1)) != "[n5 n4]" || fmt.Sprint(titles(next)) != "[n3 n2]" {
		t.Errorf("cursor pages = %v, %v", titles(page1), titles(next))
	}
	last, _ := r.List(ctx, ListParams{Limit: 2, After: next[len(next)-1].ID})
	if fmt.Sprint(titles(last)) != "[n1]" {
		t.Errorf("last page = %v", titles(last))
	}
}

func TestPartialUpdate(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	n, _ := r.Create(ctx, "title", "old content", 0)
	time.Sleep(5 * time.Millisecond)

	newContent := "new content"
	got, err := r.Update(ctx, n.ID.Hex(), nil, &newContent)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "title" || got.Content != "new content" {
		t.Errorf("partial update changed wrong fields: %+v", got)
	}
	if !got.UpdatedAt.After(n.UpdatedAt) || !got.CreatedAt.Equal(n.CreatedAt) {
		t.Errorf("timestamps: created %v updated %v", got.CreatedAt, got.UpdatedAt)
	}

	if _, err := r.Update(ctx, primitive.NewObjectID().Hex(), nil, &newContent); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: err = %v", err)
	}
	// переименование в уже занятый заголовок
	r.Create(ctx, "taken", "x", 0)
	taken := "taken"
	if _, err := r.Update(ctx, n.ID.Hex(), &taken, nil); !errors.Is(err, ErrDuplicate) {
		t.Errorf("duplicate title: err = %v", err)
	}
}

func TestDelete(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()
	n, _ := r.Create(ctx, "bye", "", 0)
	if err := r.Delete(ctx, n.ID.Hex()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ByID(ctx, n.ID.Hex()); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: err = %v", err)
	}
	if err := r.Delete(ctx, n.ID.Hex()); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: err = %v", err)
	}
}

func TestStats(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()

	empty, err := r.Stats(ctx)
	if err != nil || empty.Count != 0 {
		t.Fatalf("empty stats = %+v, %v", empty, err)
	}

	r.Create(ctx, "a", "12345", 0)      // 5 символов
	r.Create(ctx, "b", "1234567890", 0) // 10
	r.Create(ctx, "c", "Привет", 0)     // 6 символов (кириллица считается по символам, не по байтам)
	s, err := r.Stats(ctx)
	if err != nil || s.Count != 3 || s.MaxContentLength != 10 || s.AvgContentLength < 6.99 || s.AvgContentLength > 7.01 {
		t.Errorf("stats = %+v, %v", s, err)
	}
}

func TestTTLIndexAndExpiresAt(t *testing.T) {
	r, ctx := newTestRepo(t), context.Background()

	n, err := r.Create(ctx, "temp", "", 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if n.ExpiresAt == nil || n.ExpiresAt.Sub(n.CreatedAt) != 90*time.Second {
		t.Errorf("expiresAt = %v (created %v)", n.ExpiresAt, n.CreatedAt)
	}

	// TTL-индекс создан с expireAfterSeconds = 0 (удаление в момент expiresAt)
	cur, err := r.col.Indexes().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var idx []bson.M
	if err := cur.All(ctx, &idx); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, i := range idx {
		if i["name"] == "expiresAt_ttl" {
			found = true
			if i["expireAfterSeconds"] != int32(0) {
				t.Errorf("expireAfterSeconds = %v", i["expireAfterSeconds"])
			}
		}
	}
	if !found {
		t.Error("TTL index expiresAt_ttl not found")
	}
}

func TestContextTimeout(t *testing.T) {
	r := newTestRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	if _, err := r.ByID(ctx, primitive.NewObjectID().Hex()); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}
