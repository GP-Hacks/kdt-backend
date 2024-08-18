package storage

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(addr string, db int) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})
	return &RedisStorage{client: client}
}

func (s *RedisStorage) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *RedisStorage) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, expiration).Err()
}
