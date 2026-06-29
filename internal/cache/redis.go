package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr, password string, db int) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     password,
			DB:           db,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}),
	}
}

func (c *RedisCache) Get(ctx context.Context, key string, value any) error {
	payload, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: %s", apperrors.ErrCacheMiss, key)
	}
	if err != nil {
		return fmt.Errorf("%w: get %s: %v", apperrors.ErrCacheError, key, err)
	}
	if err := json.Unmarshal(payload, value); err != nil {
		return fmt.Errorf("%w: decode %s: %v", apperrors.ErrCacheError, key, err)
	}
	return nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: encode %s: %v", apperrors.ErrCacheError, key, err)
	}
	if err := c.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("%w: set %s: %v", apperrors.ErrCacheError, key, err)
	}
	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("%w: delete %s: %v", apperrors.ErrCacheError, key, err)
	}
	return nil
}
