package store

import (
	"log"
	"sync"
)

var redis_data = make(map[string]string)
var mu sync.RWMutex

func Exists(key string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := redis_data[key]
	return ok
}

func Delete(key string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := redis_data[key]; ok {
		delete(redis_data, key)
		return true
	}
	return false
}

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
