package cache

// CacheStore is an interface for a cache store.
type CacheStore interface {
    // Get retrieves a value from the cache store.
    Get(key string) (any, error)
    // Set stores a value in the cache store.
    Set(key string, value any) error
    // Delete removes a value from the cache store.
    Delete(key string) error
}