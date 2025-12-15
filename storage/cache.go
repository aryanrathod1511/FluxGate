package storage

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

type CacheEntry struct {
	Body       []byte
	Header     http.Header
	ExpiryTime time.Time
}

type lruItem struct {
	key   string
	entry CacheEntry
}

type LRUCache struct {
	capacity int
	ttl      time.Duration

	mu    sync.Mutex
	ll    *list.List
	items map[string]*list.Element
}

func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *LRUCache) Size() int {
	return c.ll.Len()
}

func (c *LRUCache) Get(key string) (CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return CacheEntry{}, false
	}

	item := elem.Value.(*lruItem)

	// TTL check
	if time.Now().After(item.entry.ExpiryTime) {
		c.ll.Remove(elem)
		delete(c.items, key)
		return CacheEntry{}, false
	}

	// move to front (most recently used)
	c.ll.MoveToFront(elem)
	return item.entry, true
}

func (c *LRUCache) Set(key string, entry CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If exists, update and move to front
	if elem, ok := c.items[key]; ok {
		c.ll.MoveToFront(elem)
		elem.Value.(*lruItem).entry = entry
		return
	}

	// If full, evict least recently used
	if c.ll.Len() >= c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			oldItem := oldest.Value.(*lruItem)
			delete(c.items, oldItem.key)
			c.ll.Remove(oldest)
		}
	}

	// Insert new
	elem := c.ll.PushFront(&lruItem{
		key:   key,
		entry: entry,
	})
	c.items[key] = elem
}
