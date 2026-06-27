package dashboard_cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"investment_monitoring/config"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DefaultTTL = 3 * time.Minute

type Cache struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

func New(rdb *redis.Client, cfg *config.Config) *Cache {
	prefix := "invmonitor"
	if cfg != nil && cfg.REDIS_PREFIX != "" {
		prefix = cfg.REDIS_PREFIX
	}
	return &Cache{
		rdb:    rdb,
		prefix: prefix,
		ttl:    DefaultTTL,
	}
}

func (c *Cache) Enabled() bool {
	return c != nil && c.rdb != nil
}

func (c *Cache) Get(ctx context.Context, key string, dest any) (bool, error) {
	if !c.Enabled() {
		return false, nil
	}

	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value any) error {
	if !c.Enabled() {
		return nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, c.ttl).Err()
}

func (c *Cache) InvalidateProject(ctx context.Context, projectID uuid.UUID) error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Del(ctx, c.SCurveKey(projectID), c.StatsKey(projectID)).Err()
}

func (c *Cache) SCurveKey(projectID uuid.UUID) string {
	return fmt.Sprintf("%s:dashboard:scurve:%s", c.prefix, projectID)
}

func (c *Cache) StatsKey(projectID uuid.UUID) string {
	return fmt.Sprintf("%s:dashboard:stats:%s", c.prefix, projectID)
}
