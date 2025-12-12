package cache

import (
	"context"
	"time"
)

type Cache interface {
	GetJSON(ctx context.Context, key string, dst any) (hit bool, err error)
	SetJSON(ctx context.Context, key string, val any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}
