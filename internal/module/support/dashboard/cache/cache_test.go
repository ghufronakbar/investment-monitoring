package dashboard_cache

import (
	"context"
	"testing"

	"investment_monitoring/config"

	"github.com/google/uuid"
)

func TestCacheDisabledWithoutRedisClient(t *testing.T) {
	cache := New(nil, &config.Config{REDIS_PREFIX: "test"})

	if cache.Enabled() {
		t.Fatal("expected cache to be disabled")
	}

	var dest map[string]string
	ok, err := cache.Get(context.Background(), "key", &dest)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss when disabled")
	}
	if err := cache.Set(context.Background(), "key", map[string]string{"a": "b"}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := cache.InvalidateProject(context.Background(), uuid.New()); err != nil {
		t.Fatalf("InvalidateProject returned error: %v", err)
	}
}

func TestCacheKeysUseConfiguredPrefix(t *testing.T) {
	projectID := uuid.New()
	cache := New(nil, &config.Config{REDIS_PREFIX: "custom"})

	if got, want := cache.SCurveKey(projectID), "custom:dashboard:scurve:"+projectID.String(); got != want {
		t.Fatalf("s-curve key = %s, want %s", got, want)
	}
	if got, want := cache.StatsKey(projectID), "custom:dashboard:stats:"+projectID.String(); got != want {
		t.Fatalf("stats key = %s, want %s", got, want)
	}
}

func TestCacheKeysUseDefaultPrefix(t *testing.T) {
	projectID := uuid.New()
	cache := New(nil, nil)

	if got, want := cache.SCurveKey(projectID), "invmonitor:dashboard:scurve:"+projectID.String(); got != want {
		t.Fatalf("s-curve key = %s, want %s", got, want)
	}
}
