/*
Package cache предоставляет реализацию универсального кэша с поддержкой различных стратегий очистки.

Пакет реализует кэш, который позволяет хранить пары "ключ-значение" с поддержкой тайм-аутов (TTL).
Кэш можно настроить для использования различных стратегий очистки:
- LRU (Least Recently Used) — удаляет наименее недавно использованные элементы.
- LFU (Least Frequently Used) — удаляет наименее часто используемые элементы.

Кэш имеет возможность автоматически очищать устаревшие элементы на основе указанного времени жизни (TTL).

Основные компоненты:

1. **Item** - представляет собой отдельный элемент в кэше.
2. **Cache** - основной тип для кэша с поддержкой различных стратегий очистки.
3. **NewCache** - функция для создания нового экземпляра кэша с заданными параметрами.
4. **Get** - метод для получения значения из кэша по ключу.
5. **Set** - метод для добавления или обновления значения в кэше.
6. **Invalidate** - метод для удаления элемента из кэша по ключу.
7. **cleanup** - метод для периодической очистки устаревших элементов.

Пример использования:

    package main

    import (
        "fmt"
        "time"
        "path/to/cache"
    )

    func main() {
        // Создаем новый кэш с вместимостью 100 элементов и стратегией очистки LRU
        c := cache.NewCache[string, int](100, "LRU", 10*time.Minute)

        // Добавляем элементы в кэш с временем жизни 5 минут
        c.Set("key1", 42)
        c.Set("key2", 100)

        // Получаем значение из кэша
        value, found := c.Get("key1")
        if found {
            fmt.Printf("Значение для 'key1': %d\n", value)
        } else {
            fmt.Println("Элемент 'key1' не найден в кэше")
        }

        // Удаляем элемент из кэша
        c.Invalidate("key1")

        // Проверяем удаление
        _, found = c.Get("key1")
        if !found {
            fmt.Println("Элемент 'key1' успешно удален")
        }
    }
*/

package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
)

// Item - структура для элемента кеша с TTL и трейсингом
type Item[K comparable, V any] struct {
	Key        K
	Value      V
	Expiration int64
}

// Cache - универсальная структура для кэша
type Cache[K comparable, V any] struct {
	capacity     int
	items        map[K]*list.Element
	evictionList *list.List
	mu           sync.Mutex
	evictionFunc func()
	ttl          time.Duration
}

// NewCache - функция для создания нового кэша
func NewCache[K comparable, V any](capacity int, evictionStrategy config.EvictionStrategy, ttl time.Duration) *Cache[K, V] {
	cache := &Cache[K, V]{
		capacity:     capacity,
		items:        make(map[K]*list.Element),
		evictionList: list.New(),
		ttl:          ttl,
	}

	switch evictionStrategy {
	case config.EvictionStrategyLRU:
		cache.evictionFunc = cache.evictLRU
	case config.EvictionStrategyLFU:
		cache.evictionFunc = cache.evictLFU
	default:
		cache.evictionFunc = cache.evictLRU
	}

	go cache.cleanup()

	return cache
}

// Get - метод для получения значения из кэша по ключу
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	const op = "cache.Cache.Get"

	span, _ := opentracing.StartSpanFromContext(context.Background(), op)
	defer span.Finish()

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, item, found := c.getElement(key)
	if found {
		c.evictionList.MoveToFront(elem)
		if item.Expiration == 0 || item.Expiration > time.Now().UnixNano() {
			return item.Value, true
		}

		// Элемент устарел
		c.evictionList.Remove(elem)
		delete(c.items, key)
	}

	var defaultValue V
	return defaultValue, false
}

// Set - метод для добавления нового значения в кэш
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) {
	const op = "cache.Cache.Set"

	span, _ := opentracing.StartSpanFromContext(context.Background(), op)
	defer span.Finish()

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	elem, item, found := c.getElement(key)
	if found {
		c.evictionList.MoveToFront(elem)
		item.Value = value
		item.Expiration = now.Add(c.ttl).UnixNano()
	} else {
		if c.evictionList.Len() >= c.capacity {
			c.evictionFunc()
		}

		expiration := now.Add(c.ttl).UnixNano()
		item := &Item[K, V]{
			Key:        key,
			Value:      value,
			Expiration: expiration,
		}
		elem := c.evictionList.PushFront(item)
		c.items[key] = elem
	}
}

// getElement метод для получения элемента и приведения его к нужному типу
func (c *Cache[K, V]) getElement(key K) (*list.Element, *Item[K, V], bool) {
	elem, found := c.items[key]
	if !found {
		return nil, nil, false
	}

	item := elem.Value.(*Item[K, V])
	return elem, item, found
}

// Invalidate - метод для инвалидации элемента кеша по ключу
func (c *Cache[K, V]) Invalidate(key K) {
	const op = "cache.Cache.Invalidate"

	span, _ := opentracing.StartSpanFromContext(context.Background(), op)
	defer span.Finish()

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, found := c.items[key]; found {
		c.evictionList.Remove(elem)
		delete(c.items, key)
	}
}

// evictLRU - метод для удаления самого старого элемента (LRU)
func (c *Cache[K, V]) evictLRU() {
	elem := c.evictionList.Back()
	if elem != nil {
		c.evictionList.Remove(elem)
		item := elem.Value.(*Item[K, V])
		delete(c.items, item.Key)
	}
}

// evictLFU - метод для удаления наименее используемого элемента (LFU)
func (c *Cache[K, V]) evictLFU() {
	var minElem *list.Element
	var minFreq int64 = 0

	for elem := c.evictionList.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*Item[K, V])
		if minFreq == 0 || item.Expiration < minFreq {
			minElem = elem
			minFreq = item.Expiration
		}
	}

	if minElem != nil {
		c.evictionList.Remove(minElem)
		item := minElem.Value.(*Item[K, V])
		delete(c.items, item.Key)
	}
}

func (c *Cache[K, V]) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()

		now := time.Now().UnixNano()
		for elem := c.evictionList.Back(); elem != nil; elem = elem.Prev() {
			item := elem.Value.(*Item[K, V])
			if item.Expiration > 0 && item.Expiration <= now {
				c.evictionList.Remove(elem)
				delete(c.items, item.Key)
			} else {
				break
			}
		}

		c.mu.Unlock()
	}
}
