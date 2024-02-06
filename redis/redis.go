package redis

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/isofinly/cachease/config"
)

type CacheConn struct {
	pool *redis.Pool
}

var Cache CacheConn
var cacheExp int

// RedisPoolInit initializes the Redis connection pool.
//
// Parameters:
//
//	redisHost string: The Redis host to connect to.
//
// Returns:
//
//	None.
func RedisPoolInit(redisHost string) {
	var err error
	cacheExp, err = strconv.Atoi(config.GetConfig("CACHE_TTL"))
	if err != nil {
		log.Println("Error while parsing cache TTL: ", err)
		cacheExp = 20 // default value
		log.Println("Using default cache TTL (20 seconds)")
	}

	Cache = CacheConn{
		pool: &redis.Pool{
			MaxIdle:     10,
			IdleTimeout: 60 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", redisHost)
			},
		},
	}
}

// ClearCache clears the entire cache
//
// Parameters:
//
//	None.
//
// Returns:
//
//	None.
func (cache *CacheConn) ClearCache() error {
	conn := cache.pool.Get()
	defer conn.Close()

	_, err := conn.Do("FLUSHALL")
	if err != nil {
		log.Println("Error while clearing cache: ", err)
		return err
	}
	log.Println("Cache cleared successfully")
	return nil
}

// SetDetails sets the details of a given key in the cache.
//
// Parameters:
//
//	key string: the key of the cache entry.
//	value interface{}: the value to be associated with the key.
//
// Returns:
//
//	error: an error if there was a problem setting the key or its expiry.
func (cache *CacheConn) SetDetails(key, value any) error {
	conn := cache.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("SETEX", key, cacheExp, value)
	if err != nil {
		log.Println("Error while setting key: ", err)
		return err
	}
	log.Println("Cache server reply on key set: ", reply)

	return nil
}

// SetDetailsWithExp sets the value of a key in the cache with a specified expiration time.
//
// Parameters:
//
//	key string: The key to set.
//	value any: The value to set.
//	expiration int: The expiration time in seconds.
//
// Returns:
//
//	error: An error, if any.
func (cache *CacheConn) SetDetailsWithExp(key string, value any, expiration int) error {
	conn := cache.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("SETEX", key, expiration, value)
	if err != nil {
		log.Println("Error while setting key: ", err)
		return err
	}
	log.Println("Cache server reply on key set: ", reply)

	return nil
}

// GetDetails retrieves the details for a given key from the cache.
//
// Parameters:
//
//	key string: the key for which to retrieve the details.
//
// Returns:
//
//	string: the details for the given key.
//	error: an error if there was a problem retrieving the details.
func (cache *CacheConn) GetDetails(key string) (string, error) {
	conn := cache.pool.Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do("GET", key))
	if err != nil {
		log.Println("An error occurred while fetching key from cache", err.Error())
		return "", err
	}
	return reply, nil
}

// GetManyDetails fetches the values of the specified keys from the cache.
//
// Parameters:
//
//	keys ...string: The keys to fetch.
//
// Returns:
//
//	(map[string]string, error): A map of key-value pairs
//	error: an error if there was a problem retrieving the details.
func (cache *CacheConn) GetManyDetails(keys ...string) (map[string]string, error) {
	conn := cache.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	values, err := redis.Strings(conn.Do("MGET", args...))
	if err != nil {
		log.Println("An error occurred while fetching keys from cache", err.Error())
		return nil, err
	}

	result := make(map[string]string)
	for i, key := range keys {
		result[key] = values[i]
	}

	return result, nil
}

// IfExistsInCache checks if a given key exists in the cache.
//
// Parameters:
//
//	key string: the key to check in the cache.
//
// Returns:
//
//	bool: true if the key exists in the cache, false otherwise.
//	error: an error if something went wrong while checking the key in the cache.
func (cache *CacheConn) IfExistsInCache(key string) (bool, error) {
	conn := cache.pool.Get()
	defer conn.Close()
	exists, err := redis.Int(conn.Do("EXISTS", key))
	if err != nil {
		log.Println("An error occurred while checking if the key exists in cache", err.Error())
		return false, err
	}

	if exists == 1 {
		log.Printf("Cache hit with key: [%s]\n", key)
		return true, nil
	}

	return false, fmt.Errorf("key doesn't exists")

}

// DeleteKey deletes a key from the cache.
//
// Parameters:
//
//	key string: the key to be deleted from the cache.
//
// Returns:
//
//	bool: true if the key was successfully deleted, false otherwise.
//	error: an error if any occurred during the deletion process.
func (cache *CacheConn) DeleteKey(key string) (bool, error) {
	conn := cache.pool.Get()
	defer conn.Close()
	_, err := redis.Int(conn.Do("DEL", key))
	if err != nil {
		log.Println("An error occurred while deleting key from cache: ", err.Error())
		return false, err
	}
	return true, nil

}

// DeleteKeys deletes the specified keys from the cache.
//
// Parameters:
//
//	keys ...string: The keys to delete.
//
// Returns:
//
//	int: The number of keys deleted.
//	error: an error if there was a problem retrieving the details.
func (cache *CacheConn) DeleteKeys(keys ...string) (int, error) {
	conn := cache.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	count, err := redis.Int(conn.Do("DEL", args...))
	if err != nil {
		log.Println("An error occurred while deleting keys from cache: ", err.Error())
		return 0, err
	}
	return count, nil
}
