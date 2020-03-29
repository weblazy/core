package collection

import (
	"container/list"
	"sync"
	"time"
)

const slots = 300

var emptyLruCache = emptyLru{}

type (
	CacheOption func(cache *Cache)

	Cache struct {
		lock        sync.RWMutex
		data        map[string]interface{}
		evicts      *list.List
		expire      time.Duration
		timingWheel *TimingWheel
		lruCache    lru
	}
)

func NewCache(expire time.Duration, opts ...CacheOption) (*Cache, error) {
	cache := &Cache{
		data:     make(map[string]interface{}),
		expire:   expire,
		lruCache: emptyLruCache,
	}

	for _, opt := range opts {
		opt(cache)
	}

	timingWheel, err := NewTimingWheel(time.Second, slots, func(k, v interface{}) {
		key, ok := k.(string)
		if !ok {
			return
		}

		cache.Del(key)
	})
	if err != nil {
		return nil, err
	}

	cache.timingWheel = timingWheel

	return cache, nil
}

func (c *Cache) Del(key string) {
	c.lock.Lock()
	delete(c.data, key)
	c.lruCache.remove(key)
	c.lock.Unlock()
	c.timingWheel.RemoveTimer(key)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.lock.RLock()
	value, ok := c.data[key]
	c.lock.RUnlock()

	return value, ok
}

func (c *Cache) Set(key string, value interface{}) {
	c.lock.Lock()
	_, ok := c.data[key]
	c.data[key] = value
	c.lruCache.add(key)
	c.lock.Unlock()

	if ok {
		c.timingWheel.MoveTimer(key, c.expire)
	} else {
		c.timingWheel.SetTimer(key, value, c.expire)
	}
}

func (c *Cache) onEvict(key string) {
	// already locked
	delete(c.data, key)
	c.timingWheel.RemoveTimer(key)
}

func WithLimit(limit int) CacheOption {
	return func(cache *Cache) {
		if limit > 0 {
			cache.lruCache = newKeyLru(limit, cache.onEvict)
		}
	}
}

type (
	lru interface {
		add(key string)
		remove(key string)
	}

	emptyLru struct{}

	keyLru struct {
		limit    int
		evicts   *list.List
		elements map[string]*list.Element
		onEvict  func(key string)
	}
)

func (elru emptyLru) add(key string) {
}

func (elru emptyLru) remove(key string) {
}

func newKeyLru(limit int, onEvict func(key string)) *keyLru {
	return &keyLru{
		limit:    limit,
		evicts:   list.New(),
		elements: make(map[string]*list.Element),
		onEvict:  onEvict,
	}
}

func (klru *keyLru) add(key string) {
	if elem, ok := klru.elements[key]; ok {
		klru.evicts.MoveToFront(elem)
		return
	}

	// Add new item
	elem := klru.evicts.PushFront(key)
	klru.elements[key] = elem

	// Verify size not exceeded
	if klru.evicts.Len() > klru.limit {
		klru.removeOldest()
	}
}

func (klru *keyLru) remove(key string) {
	if elem, ok := klru.elements[key]; ok {
		klru.removeElement(elem)
	}
}

func (klru *keyLru) removeOldest() {
	elem := klru.evicts.Back()
	if elem != nil {
		klru.removeElement(elem)
	}
}

func (klru *keyLru) removeElement(e *list.Element) {
	klru.evicts.Remove(e)
	key := e.Value.(string)
	delete(klru.elements, key)
	klru.onEvict(key)
}
