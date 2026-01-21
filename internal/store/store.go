package store

import (
	"log"
	"sync"
	"time"
)

var redis_data = make(map[string]string)
var expiry = make(map[string]time.Time)
var mu sync.RWMutex

func Exists(key string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := redis_data[key]
	if ok {
		if data, ok := expiry[key]; ok {
			ttl := int(time.Until(data).Truncate(time.Second).Seconds())
			if ttl <= 0 {
				return false
			}
		}
	}
	return ok
}

func Delete(key string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := redis_data[key]; ok {
		delete(redis_data, key)
		delete(expiry, key)
		return true
	}
	return false
}

func Get(key string) (string, bool) {
	mu.RLock()
	defer mu.RUnlock()
	data, ok := redis_data[key]
	if ok {
		if data, ok := expiry[key]; ok {
			ttl := int(time.Until(data).Truncate(time.Second).Seconds())
			if ttl <= 0 {
				return "", false
			}
		}
		return data, true
	}
	return "", false
}

func SetWithExpiry(key string, value string, seconds time.Time) {
	mu.Lock()
	defer mu.Unlock()
	redis_data[key] = value
	expiry[key] = seconds
}

func GetTTL(key string) (time.Time, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if data, ok := expiry[key]; ok {
		return data, true
	}
	return time.Time{}, false
}

func SetExpire(key string, seconds time.Time) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := redis_data[key]; ok {
		expiry[key] = seconds
		return true
	}
	return false
}

func Set(key string, value string) bool {
	mu.Lock()
	defer mu.Unlock()
	if redis_data == nil {
		log.Print("DB is null")
		return false
	}

	redis_data[key] = value
	return true
}

// Clean data that is expired
func CleanUp() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		mu.Lock()
		for key, exp := range expiry {
			if !exp.After(now) {
				delete(expiry, key)
				delete(redis_data, key)
			}
		}
		mu.Unlock()
	}
}
