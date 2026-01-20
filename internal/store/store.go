package store

import (
	"log"
	"sync"
)

var redis_data = make(map[string]string)
var mu sync.RWMutex

func Get(key string) (string, bool) {
	mu.RLock()
	defer mu.RUnlock()
	data, ok := redis_data[key]
	if ok {
		return data, true
	}
	return "", false
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
