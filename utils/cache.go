package utils

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// Usage：
// utils.CacheCreate(10, 60)
// utils.CacheSetKey("name", "fish", 0)
// logger.Println("set key")

// cnt := 0
// for {
// 	cnt += 1
// 	name, ok := utils.CacheGetKey("name")
// 	if ok {
// 		logger.Println("idx=", cnt, "get key:", name)
// 	} else {
// 		logger.Println("key expire.")
// 		break
// 	}
// 	utils.SleepSec(1)
// }

var c *cache.Cache

// 创建一个带有默认缓存过期时间和清理间隔时间的缓存对象
// 时间单位为秒数
func CacheCreate(expireSecs int, cleanSecs int) {
	if c == nil {
		c = cache.New(time.Duration(expireSecs)*time.Second, time.Duration(cleanSecs)*time.Second)
	}
}

// set key
// Add an item to the cache, replacing any existing item.
// If the duration is 0 (DefaultExpiration), the cache's default expiration time is used.
// If it is -1 (NoExpiration), the item never expires.
func CacheSetKey(key string, val interface{}, d time.Duration) {
	if c != nil {
		c.Set(key, val, d)
	}
}

// get key
func CacheGetKey(key string) (interface{}, bool) {
	if c != nil {
		return c.Get(key)
	}
	return nil, false
}
