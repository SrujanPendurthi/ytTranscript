// Package ratelimit provides a small Redis-backed fixed-window request limiter.
// It is domain-agnostic; callers supply the key, limit, and window.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Limiter {
	return &Limiter{rdb: rdb}
}

// Allow increments the counter for key within the current fixed window and
// reports whether the request is permitted. When denied, it also returns how
// long until the window resets (for a Retry-After header). It fails open if
// Redis is unavailable so a cache blip never takes the API down.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration) {
	if limit <= 0 {
		return true, 0
	}
	now := time.Now()
	bucket := now.Truncate(window).Unix()
	rkey := fmt.Sprintf("rl:%s:%d", key, bucket)

	count, err := l.rdb.Incr(ctx, rkey).Result()
	if err != nil {
		return true, 0 // fail open
	}
	if count == 1 {
		l.rdb.Expire(ctx, rkey, window)
	}
	if count > int64(limit) {
		reset := now.Truncate(window).Add(window)
		return false, time.Until(reset)
	}
	return true, 0
}
