package store

import (
	"log"
	"sync"
	"time"
)

var redis_data = make(map[string]string)
var expiry = make(map[string]time.Time)
var list_data = make(map[string][]string)

type inner_hash_data map[string]string

var hash_data = make(map[string]inner_hash_data)
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
	mu.Lock()
	defer mu.Unlock()
	data := list_data[key]
	data = append([]string{value}, data...)
	list_data[key] = data
	return len(data)
}

func RPush(key string, value string) int {
	mu.Lock()
	defer mu.Unlock()
	data := list_data[key]
	data = append(data, value)
	list_data[key] = data
	return len(data)
}

func LPop(key string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
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
	mu.Lock()
	defer mu.Unlock()
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
	mu.Lock()
	defer mu.Unlock()
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
	mu.RLock()
	defer mu.RUnlock()
	if data, ok := list_data[key]; ok {
		return len(data)
	}
	return -1
}

// Hash Functions
func HSet(key string, field string, value string) int {
	mu.Lock()
	defer mu.Unlock()

	// ensure inner map exists
	if _, ok := hash_data[key]; !ok {
		hash_data[key] = make(inner_hash_data)
	}

	if _, existed := hash_data[key][field]; existed {
		hash_data[key][field] = value
		return 0
	}
	hash_data[key][field] = value
	return 1
}

func HGet(key string, field string) (string, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := hash_data[key]; ok {
		if response, ok2 := m[field]; ok2 {
			return response, true
		}
	}
	return "", false
}

func HDel(key string, field string) int {
	mu.Lock()
	defer mu.Unlock()
	if m, ok := hash_data[key]; ok {
		if _, ok2 := m[field]; ok2 {
			delete(m, field)
			return 1
		}
	}
	return 0
}

func HGetAll(key string) []string {
	mu.RLock()
	defer mu.RUnlock()
	response := make([]string, 0)
	if m, ok := hash_data[key]; ok {
		for k, v := range m {
			response = append(response, k)
			response = append(response, v)
		}
	}
	return response
}

func HKeys(key string) []string {
	mu.RLock()
	defer mu.RUnlock()
	response := make([]string, 0)
	if m, ok := hash_data[key]; ok {
		for field := range m {
			response = append(response, field)
		}
	}
	return response
}

func HLen(key string) int {
	mu.RLock()
	defer mu.RUnlock()
	length := 0
	m, ok := hash_data[key]
	if !ok {
		return length
	}
	for k, v := range m {
		if k == "" || v == "" {
			length++
		} else {
			length += 2
		}
	}
	return length
}
