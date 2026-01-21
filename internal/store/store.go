package store

import (
	"log"
	"sync"
	"time"
)

var redis_data = make(map[string]string)
var expiry = make(map[string]time.Time)
var list_data = make(map[string][]string)
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

// LIST Fucntions

func LPush(key string, value string) int {
	data := list_data[key]
	data = append([]string{value}, data...)
	list_data[key] = data
	return len(data)
}

func RPush(key string, value string) int {
	data := list_data[key]
	data = append(data, value)
	list_data[key] = data
	return len(data)
}

func LPop(key string) (string, bool) {
	if len(list_data) <= 0 {
		return "", false
	}
	if data, ok := list_data[key]; ok {
		response := data[0]
		log.Print("Deleting from right: ", response)
		list_data[key] = data[1:]
		return response, true
	}

	return "", false
}

func RPop(key string) (string, bool) {
	if len(list_data) <= 0 {
		return "", false
	}
	if data, ok := list_data[key]; ok {
		response := data[len(data)-1]
		log.Print("Deleting from right: ", response)
		list_data[key] = data[:len(data)-1]
		return response, true
	}

	return "", false
}

func LRange(key string, start int, stop int) ([]string, bool) {
	if start < 0 || stop > len(list_data) {
		return nil, false
	}
	if start == 0 && stop == -1 {
		return list_data[key], true
	}
	response := make(map[string][]string)
	i := 0
	if _, ok := list_data[key]; !ok {
		return nil, false
	}
	for _, value := range list_data[key] {
		if i >= start && i <= stop {
			response[key] = append(response[key], value)
		}
		i++
	}
	return response[key], true
}

func LLen(key string) int {
	if data, ok := list_data[key]; ok {
		return len(data)
	}
	return -1
}
