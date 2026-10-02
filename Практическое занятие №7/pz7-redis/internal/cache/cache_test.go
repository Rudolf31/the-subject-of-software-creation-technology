package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	c := New(addr)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("Redis недоступен: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// uniqueKey исключает пересечения между тестами и с реальными данными.
func uniqueKey(t *testing.T) string {
	return fmt.Sprintf("test:%s:%d", t.Name(), time.Now().UnixNano())
}

func TestSetGet(t *testing.T) {
	c, ctx, key := newTestCache(t), context.Background(), ""
	key = uniqueKey(t)
	t.Cleanup(func() { c.Del(ctx, key) })

	if err := c.Set(ctx, key, "hello", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, key)
	if err != nil || got != "hello" {
		t.Fatalf("Get = %q, %v", got, err)
	}
}

func TestGetMissingKey(t *testing.T) {
	c := newTestCache(t)
	if _, err := c.Get(context.Background(), uniqueKey(t)); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if _, err := c.TTL(context.Background(), uniqueKey(t)); !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL err = %v, want ErrNotFound", err)
	}
}

func TestTTLCountsDown(t *testing.T) {
	c, ctx := newTestCache(t), context.Background()
	key := uniqueKey(t)
	t.Cleanup(func() { c.Del(ctx, key) })

	c.Set(ctx, key, "v", 10*time.Second)
	ttl, err := c.TTL(ctx, key)
	if err != nil || ttl <= 8*time.Second || ttl > 10*time.Second {
		t.Fatalf("TTL = %v, %v; want ~10s", ttl, err)
	}
}

func TestKeyExpires(t *testing.T) {
	c, ctx := newTestCache(t), context.Background()
	key := uniqueKey(t)

	c.Set(ctx, key, "short-lived", time.Second)
	if _, err := c.Get(ctx, key); err != nil {
		t.Fatalf("key should exist right after Set: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)

	// «протухший» ключ ведёт себя как отсутствующий
	if _, err := c.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after TTL: err = %v, want ErrNotFound", err)
	}
	if _, err := c.TTL(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL after expiry: err = %v, want ErrNotFound", err)
	}
}

func TestNoExpiry(t *testing.T) {
	c, ctx := newTestCache(t), context.Background()
	key := uniqueKey(t)
	t.Cleanup(func() { c.Del(ctx, key) })

	c.Set(ctx, key, "forever", 0)
	if ttl, err := c.TTL(ctx, key); err != nil || ttl != NoExpiry {
		t.Errorf("TTL = %v, %v; want NoExpiry", ttl, err)
	}
}

func TestDel(t *testing.T) {
	c, ctx := newTestCache(t), context.Background()
	key := uniqueKey(t)

	c.Set(ctx, key, "x", time.Minute)
	if ok, err := c.Del(ctx, key); !ok || err != nil {
		t.Fatalf("Del = %v, %v", ok, err)
	}
	if ok, _ := c.Del(ctx, key); ok {
		t.Error("second Del should report false")
	}
}

func TestUnavailableRedisReturnsError(t *testing.T) {
	c := New("127.0.0.1:1") // порт, на котором Redis нет
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := c.Get(ctx, "k"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want connection error (not ErrNotFound)", err)
	}
}
