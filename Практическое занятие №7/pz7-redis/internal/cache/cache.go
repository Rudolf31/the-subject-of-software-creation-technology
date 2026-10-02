package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound — ключа нет (не было или «протух» по TTL).
var ErrNotFound = errors.New("key not found")

// NoExpiry возвращает TTL для ключа без срока жизни.
const NoExpiry time.Duration = -1

type Cache struct {
	rdb *redis.Client
}

func New(addr string) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	return &Cache{rdb: rdb}
}

func (c *Cache) Close() error { return c.rdb.Close() }

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Set сохраняет значение; ttl == 0 означает «без срока жизни».
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Get возвращает значение или ErrNotFound, если ключа нет.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return val, err
}

// TTL возвращает оставшееся время жизни ключа.
// Нет ключа → ErrNotFound; ключ без срока жизни → NoExpiry.
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// go-redis возвращает «сырые» -2 и -1 как Duration, а не как секунды
	switch d {
	case -2:
		return 0, ErrNotFound
	case -1:
		return NoExpiry, nil
	}
	return d, nil
}

// Del удаляет ключ и сообщает, существовал ли он.
func (c *Cache) Del(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Del(ctx, key).Result()
	return n > 0, err
}
