package main

import (
	"fmt"
	"os"

	"github.com/isofinly/cachease/config"
	memory "github.com/isofinly/cachease/memory"
	"github.com/isofinly/cachease/redis"
)

var c *memory.Cache

func main() {
	production := os.Getenv("PRODUCTION") == ""
	config.SetupConfig(production)

	redis.RedisPoolInit(os.Getenv("REDIS_HOST"))
	c, _ = memory.NewCache(1000)

	c.Put("test", "aboba")
	val, _ := c.Get("test")
	fmt.Printf("val: %s\n", val)

	redis.Cache.SetDetails("test2", "aboba2")
	redis.Cache.SetDetails("test3", "aboba3")
	res, _ := redis.Cache.GetManyDetails("test2", "test3")
	fmt.Printf("res: %v\n", res)
}
