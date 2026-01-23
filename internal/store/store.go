package store

import (
	"sync"
	"time"
)

var Redis_data = make(map[string]string)
var Expiry = make(map[string]time.Time)
var List_data = make(map[string][]string)

type inner_Hash_data map[string]string
type inner_Set_data map[string]bool

var Hash_data = make(map[string]inner_Hash_data)
var Set_data = make(map[string]inner_Set_data)
var mu sync.RWMutex

func Exists(key string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := Redis_data[key]
	if ok {
		if data, ok := Expiry[key]; ok {
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
	if _, ok := Redis_data[key]; ok {
		delete(Redis_data, key)
		delete(Expiry, key)
		return true
	}
	return false
}

func Get(key string) (string, bool) {
	mu.RLock()
	defer mu.RUnlock()
	data, ok := Redis_data[key]
	if ok {
		if data, ok := Expiry[key]; ok {
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
	Redis_data[key] = value
	Expiry[key] = seconds
}

func GetTTL(key string) (time.Time, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if data, ok := Expiry[key]; ok {
		return data, true
	}
	return time.Time{}, false
}

func SetExpire(key string, seconds time.Time) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := Redis_data[key]; ok {
		Expiry[key] = seconds
		return true
	}
	return false
}

func Set(key string, value string) bool {
	mu.Lock()
	defer mu.Unlock()
	if Redis_data == nil {
		return false
	}

	Redis_data[key] = value
	return true
}

// Clean data that is expired
func CleanUp() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		mu.Lock()
		for key, exp := range Expiry {
			if !exp.After(now) {
				delete(Expiry, key)
				delete(Redis_data, key)
			}
		}
		mu.Unlock()
	}
}

// LIST Fucntions

func LPush(key string, value string) int {
	mu.Lock()
	defer mu.Unlock()
	data := List_data[key]
	data = append([]string{value}, data...)
	List_data[key] = data
	return len(data)
}

func RPush(key string, value string) int {
	mu.Lock()
	defer mu.Unlock()
	data := List_data[key]
	data = append(data, value)
	List_data[key] = data
	return len(data)
}

func LPop(key string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	if len(List_data) <= 0 {
		return "", false
	}
	if data, ok := List_data[key]; ok {
		response := data[0]
		List_data[key] = data[1:]
		return response, true
	}

	return "", false
}

func RPop(key string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	if len(List_data) <= 0 {
		return "", false
	}
	if data, ok := List_data[key]; ok {
		response := data[len(data)-1]
		List_data[key] = data[:len(data)-1]
		return response, true
	}

	return "", false
}

func LRange(key string, start int, stop int) ([]string, bool) {
	mu.Lock()
	defer mu.Unlock()
	if start < 0 || stop > len(List_data[key]) {
		return nil, false
	}
	if start == 0 && stop <= -1 {
		return List_data[key], true
	}
	response := make(map[string][]string)
	i := 0
	if _, ok := List_data[key]; !ok {
		return nil, false
	}
	for _, value := range List_data[key] {
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
	if data, ok := List_data[key]; ok {
		return len(data)
	}
	return 0
}

// Hash Functions
func HSet(key string, field string, value string) int {
	mu.Lock()
	defer mu.Unlock()

	// ensure inner map exists
	if _, ok := Hash_data[key]; !ok {
		Hash_data[key] = make(inner_Hash_data)
	}

	if _, existed := Hash_data[key][field]; existed {
		Hash_data[key][field] = value
		return 0
	}
	Hash_data[key][field] = value
	return 1
}

func HGet(key string, field string) (string, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := Hash_data[key]; ok {
		if response, ok2 := m[field]; ok2 {
			return response, true
		}
	}
	return "", false
}

func HDel(key string, field string) int {
	mu.Lock()
	defer mu.Unlock()
	if m, ok := Hash_data[key]; ok {
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
	if m, ok := Hash_data[key]; ok {
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
	if m, ok := Hash_data[key]; ok {
		for field := range m {
			response = append(response, field)
		}
	}
	return response
}

func HLen(key string) int {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := Hash_data[key]
	if !ok {
		return 0
	}
	return len(Hash_data[key])
}

// SETS (unordered) Functions

func SAdd(key string, value string) int {
	mu.Lock()
	defer mu.Unlock()
	// ensure inner map exists
	if _, ok := Set_data[key]; !ok {
		Set_data[key] = make(inner_Set_data)
	}
	if _, ok := Set_data[key]; ok {
		if _, ok := Set_data[key][value]; ok {
			Set_data[key][value] = true
			return 0
		}
		Set_data[key][value] = true
		return 1
	}
	Set_data[key][value] = true
	return 1
}

func SRem(key string, value string) int {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := Set_data[key][value]; ok {
		delete(Set_data[key], value)
		return 1
	}
	return 0
}

func SMembers(key string) []string {
	mu.RLock()
	defer mu.RUnlock()
	response := make([]string, 0)
	for value, ok := range Set_data[key] {
		if ok {
			response = append(response, value)
		}
	}
	return response
}

func SIsMember(key string, member string) int {
	mu.RLock()
	defer mu.RUnlock()
	if _, ok := Set_data[key][member]; ok {
		return 1
	}
	return 0
}

func SCard(key string) int {
	mu.RLock()
	defer mu.RUnlock()
	response := 0

	for _, ok := range Set_data[key] {
		if ok {
			response++
		}
	}
	return response
}

// SnapShot (to avoid slow write operation and save data later, user won't be stopped)
func GetSnapshot() (
	map[string]string,
	map[string]time.Time,
	map[string][]string,
	map[string]map[string]string,
	map[string]map[string]bool,
) {
	mu.RLock()
	defer mu.RUnlock()

	// Strings
	redis_data := make(map[string]string)
	for k, v := range Redis_data {
		redis_data[k] = v
	}

	// Expiry
	expiry := make(map[string]time.Time)
	for k, v := range Expiry {
		expiry[k] = v
	}

	// Lists
	lists := make(map[string][]string)
	for k, v := range List_data {
		lists[k] = v
	}

	// Hash
	hashes := make(map[string]map[string]string)
	for k, v := range Hash_data {
		hashes[k] = make(map[string]string)
		for k1, v1 := range v {
			hashes[k][k1] = v1
		}
	}

	// Sets
	sets := make(map[string]map[string]bool)
	for k, v := range Set_data {
		sets[k] = make(map[string]bool)
		for k1, v1 := range v {
			sets[k][k1] = v1
		}
	}

	return redis_data, expiry, lists, hashes, sets
}
