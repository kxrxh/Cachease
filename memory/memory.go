package memory

import (
	"math/bits"
	"runtime"
	"sync"
	"time"
)

const (
	// Const DEFAULT_EXPIRATION defines the default ttl time period for all keys
	DEFAULT_EXPIRATION = 10 * time.Minute

	// Const DEFAULT_CLEAN_DURATION defines the default period of the auto clean
	DEFAULT_CLEAN_DURATION = 10 * time.Minute

	// Const DEFAULT_CAP defines the default size of the cache
	DEFAULT_CAP = 1024

	// Const DEFAULT_LRU_CLEAN_SIZE defines the default number of keys that are cleaned during auto clean
	DEFAULT_LRU_CLEAN_SIZE = 20
)

// Cache defines the structure of the Cache.
type Cache struct {
	defaultExpiration time.Duration
	elements          map[string]Elem
	capacity          int64
	size              int64
	lock              *sync.RWMutex
	pool              *sync.Pool
	cleaner           *cleaner
}

// Elem defines the item of the Cache value.
// It contains K as Key, V as value, Expiration and LastHit
type Elem struct {
	K          string
	V          any
	Expiration int64
	LastHit    int64
}

type cleaner struct {
	Interval time.Duration
	stop     chan bool
}

// Get retrieves the value associated with the given key from the cache.
//
// If the key is not found in the cache, nil is returned.
//
// Parameters:
//
//	k (string): The key to retrieve.
//
// Returns:
//
//	(interface{}, error): The value associated with the key, or nil if the key is not found.
func (c *Cache) Get(key string) (value any, err error) {
	ele := c.pool.Get()
	if item, ok := ele.(Elem); ok {
		if item.K == key {
			return item.V, nil
		}
	}
	expire := time.Now().Add(DEFAULT_EXPIRATION).UnixNano()
	lastHit := time.Now().UnixNano()
	c.lock.RLock()
	defer c.lock.RUnlock()
	if ele, ok := c.elements[key]; ok {
		ele.Expiration = expire
		ele.LastHit = lastHit
		return ele.V, nil
	}
	return nil, nil
}

// Put an element into the cache.
//
// Parameters:
//
//	key (string): The key of the element to put.
//	value (interface{}): The value of the element to put.
//
// Returns:
//
//	(error): An error if one occurred.
func (c *Cache) Put(key string, value any) error {
	expire := time.Now().Add(DEFAULT_EXPIRATION).UnixNano()
	lastHit := time.Now().UnixNano()
	if c.size+1 > c.capacity {
		// LRU kicks in
		if err := c.removeLeastVisited(); err != nil {
			return err
		}
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	if found := c.update(key, value, expire, lastHit); found {
		return nil
	}

	ele := Elem{
		V:          value,
		Expiration: expire,
		LastHit:    lastHit,
	}
	c.pool.Put(&ele)
	c.elements[key] = ele
	c.size = c.size + 1
	return nil
}

func (c *Cache) update(k string, v interface{}, expire int64, lastHit int64) bool {
	if ele, ok := c.elements[k]; ok {
		ele.V = v
		ele.Expiration = expire
		ele.LastHit = lastHit
		return true
	}
	return false
}

func (c *Cache) removeLeastVisited() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var lastTime int64 = 1<<(bits.UintSize-1) - 1 // MaxInt
	t := time.Now().UnixNano()
	lastItems := make([]string, DEFAULT_LRU_CLEAN_SIZE)
	liCount := 0
	full := false

	for k, v := range c.elements {
		if v.Expiration > t { // not expiring
			atime := v.LastHit
			if !full || atime < lastTime {
				lastTime = atime
				if liCount < DEFAULT_LRU_CLEAN_SIZE {
					lastItems[liCount] = k
					liCount++
				} else {
					lastItems[0] = k
					liCount = 1
					full = true
				}
			}
		}
	}

	for i := 0; i < len(lastItems) && lastItems[i] != ""; i++ {
		lastName := lastItems[i]
		delete(c.elements, lastName)
	}
	return nil
}

// Remove the element with the given key from the cache.
//
// Parameters:
//
//	key (string): The key of the element to remove.
//
// Returns:
//
//	(bool, error): Whether the element was found and removed, and an error if one occurred.
func (c *Cache) Remove(key string) (isFound bool, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	v := c.pool.Get()
	if v != nil && v.(Elem).K != key {
		c.pool.Put(v)
	}
	for k := range c.elements {
		if k == key {
			delete(c.elements, key)
			return true, nil
		}
	}
	return false, nil
}

// Flush cleans up the cache
func (c *Cache) Flush() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pool.Get()
	c.elements = make(map[string]Elem, DEFAULT_CAP)
	return nil
}

// RemoveExpired triggers the a clean for expired keys
//
// It is thread-safe.
func (c *Cache) RemoveExpired() {
	now := time.Now().UnixNano()
	c.lock.Lock()
	defer c.lock.Unlock()
	for k, v := range c.elements {
		if v.Expiration > 0 && now > v.Expiration {
			_, _ = c.Remove(k)
		}
	}
}

// Run cleaning job
func (cl *cleaner) Run(c *Cache) {
	ticker := time.NewTicker(cl.Interval)
	for {
		select {
		case <-ticker.C:
			c.RemoveExpired()
		case <-cl.stop:
			ticker.Stop()
			return
		}
	}
}

func stopCleaner(c *Cache) {
	c.cleaner.stop <- true
}

// NewCache creates a new cache with the specified parameters.
//
// Parameters:
//
//	params ...int: The parameters for the cache.
//
// Returns:
//
//	(*Cache, error): The cache and an error, if any.
func NewCache(params ...int) (*Cache, error) {
	if len(params) > 0 {
		return newCache(int64(params[0]), DEFAULT_EXPIRATION, DEFAULT_CLEAN_DURATION)
	}
	return newCache(DEFAULT_CAP, DEFAULT_EXPIRATION, DEFAULT_CLEAN_DURATION)
}

// newCache creates a new cache with the specified parameters.
//
// Parameters:
//
//	cap int64: The capacity of the cache.
//	expiration time.Duration: The expiration time for the cache.
//	clean_duration time.Duration: The clean duration for the cache.
//
// Returns:
//
//	(*Cache, error): The cache and an error, if any.
func newCache(cap int64, expiration time.Duration, clean_duration time.Duration) (*Cache, error) {
	c := &Cache{
		defaultExpiration: expiration,
		elements:          make(map[string]Elem, cap),
		capacity:          cap,
		lock:              new(sync.RWMutex),
		cleaner: &cleaner{
			Interval: clean_duration,
			stop:     make(chan bool),
		},
		pool: &sync.Pool{},
	}

	go c.cleaner.Run(c)
	runtime.SetFinalizer(c, stopCleaner)
	return c, nil
}
