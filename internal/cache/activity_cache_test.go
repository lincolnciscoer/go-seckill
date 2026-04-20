package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"go-seckill/internal/repository"
)

func newTestActivityCache(t *testing.T) (*ActivityCache, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})

	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})

	return NewActivityCache(client), server
}

func TestActivityCacheEmptyMarkerPreventsPenetration(t *testing.T) {
	cache, _ := newTestActivityCache(t)

	if err := cache.SetActivityEmpty(context.Background(), 404); err != nil {
		t.Fatalf("set empty activity cache: %v", err)
	}

	activity, hit, err := cache.GetActivityDetail(context.Background(), 404)
	if err != nil {
		t.Fatalf("get activity detail: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit for empty marker")
	}
	if activity != nil {
		t.Fatalf("expected nil activity for empty marker, got %#v", activity)
	}
}

func TestActivityCacheTTLHasJitter(t *testing.T) {
	cache, server := newTestActivityCache(t)

	err := cache.SetActivityDetail(context.Background(), repository.ActivityView{
		ID:             1,
		ProductID:      2,
		Name:           "test",
		AvailableStock: 10,
	})
	if err != nil {
		t.Fatalf("set activity detail: %v", err)
	}

	ttl := server.TTL("seckill:activity:1")
	if ttl < defaultCacheTTL || ttl > defaultCacheTTL+defaultCacheTTLJitter {
		t.Fatalf("expected ttl in [%v, %v], got %v", defaultCacheTTL, defaultCacheTTL+defaultCacheTTLJitter, ttl)
	}

	stockTTL := server.TTL("seckill:stock:1")
	if stockTTL < defaultCacheTTL || stockTTL > defaultCacheTTL+defaultCacheTTLJitter {
		t.Fatalf("expected stock ttl in [%v, %v], got %v", defaultCacheTTL, defaultCacheTTL+defaultCacheTTLJitter, stockTTL)
	}
}

func TestTTLWithJitterAlwaysAtLeastBase(t *testing.T) {
	for i := 0; i < 100; i++ {
		ttl := ttlWithJitter(time.Minute, 30*time.Second)
		if ttl < time.Minute || ttl > time.Minute+30*time.Second {
			t.Fatalf("ttl out of range: %v", ttl)
		}
	}
}
