package cache

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
)

type OrderCache struct {
	cache *Cache[int64, *domain.Order]
}

func NewOrderCache(capacity int, evictionStrategy config.EvictionStrategy, ttl time.Duration) *OrderCache {
	return &OrderCache{
		cache: NewCache[int64, *domain.Order](capacity, evictionStrategy, ttl),
	}
}

func (oc *OrderCache) Set(ctx context.Context, key int64, value *domain.Order) {
	const op = "storage.cache.OrderCache.Set"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("cache.key", key)

	oc.cache.Set(ctx, key, value)
}

func (oc *OrderCache) Get(ctx context.Context, key int64) (*domain.Order, bool) {
	const op = "storage.cache.OrderCache.Get"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("cache.key", key)

	return oc.cache.Get(ctx, key)
}
