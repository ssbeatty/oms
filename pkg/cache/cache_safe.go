package cache

import (
	"sync"
)

// Cache is a thread-safe fixed size LRU cache.
type Cache struct {
	lru  *Lru
	lock sync.RWMutex
}

// NewCache creates an LRU of the given size.
func NewCache(size int) (c *Cache) {
	return &Cache{
		lru:  New(size),
		lock: sync.RWMutex{},
	}
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *Cache) Add(key Key, value interface{}) {
	c.lock.Lock()
	c.lru.Add(key, value)
	c.lock.Unlock()
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.lru.Get(key)
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key interface{}) {
	c.lock.Lock()
	c.lru.Remove(key)
	c.lock.Unlock()
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	c.lock.Lock()
	c.lru.RemoveOldest()
	c.lock.Unlock()
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *Cache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lru.Keys()
}

// Length returns the number of items in the cache.
func (c *Cache) Length() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lru.Len()
}

// Clear is used to completely clear the cache.
func (c *Cache) Clear() {
	c.lock.Lock()
	c.lru.Clear()
	c.lock.Unlock()
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *Cache) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lru.Contains(key)
}
