package transcript

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyPrefix = "tf:transcript:"
	cacheTTL       = 24 * time.Hour
)

type cacheLayer struct {
	rdb *redis.Client
}

func newCacheLayer(rdb *redis.Client) *cacheLayer {
	return &cacheLayer{rdb: rdb}
}

func (c *cacheLayer) Get(ctx context.Context, videoID string) (*Result, error) {
	raw, err := c.rdb.Get(ctx, cacheKeyPrefix+videoID).Bytes()
	if err != nil {
		return nil, err // includes redis.Nil on cache miss
	}
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}
	r.Source = "cache"
	return &r, nil
}

func (c *cacheLayer) Set(ctx context.Context, videoID string, r *Result) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, cacheKeyPrefix+videoID, b, cacheTTL).Err()
}
