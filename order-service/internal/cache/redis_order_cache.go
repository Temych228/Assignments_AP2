package cache

import (
	"context"
	"encoding/json"
	"time"

	"order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, bool, error)
	Set(ctx context.Context, order *domain.Order, ttl time.Duration) error
	Delete(ctx context.Context, id string) error
}

type RedisOrderCache struct {
	client *redis.Client
	prefix string
}

func NewRedisOrderCache(client *redis.Client) *RedisOrderCache {
	return &RedisOrderCache{
		client: client,
		prefix: "order:",
	}
}

func (c *RedisOrderCache) key(id string) string {
	return c.prefix + id
}

func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, bool, error) {
	val, err := c.client.Get(ctx, c.key(id)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, false, err
	}

	return &order, true, nil
}

func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order, ttl time.Duration) error {
	b, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(order.ID), b, ttl).Err()
}

func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, c.key(id)).Err()
}
