// Package store is the thread-safe, in-memory heart of LiteKV. It owns every
// data structure (strings, lists, hashes, sets) and the TTL bookkeeping behind
// them. All state lives inside the Store struct there are no package-level
// globals — so the data can be constructed, injected and tested in isolation.
package store

import (
	"sync"
	"time"
)

// Store holds all key spaces guarded by a single RWMutex. Each data type lives
// in its own map so that commands never collide across types.
type Store struct {
	mu      sync.RWMutex
	strings map[string]string            // string keys (GET/SET)
	expiry  map[string]time.Time         // per-key TTL deadlines
	lists   map[string][]string          // list keys (LPUSH/RPUSH/...)
	hashes  map[string]map[string]string // hash keys (HSET/HGET/...)
	sets    map[string]map[string]bool   // set keys (SADD/SREM/...)
}

// New builds an empty Store with every key space initialised.
func New() *Store {
	return &Store{
		strings: make(map[string]string),
		expiry:  make(map[string]time.Time),
		lists:   make(map[string][]string),
		hashes:  make(map[string]map[string]string),
		sets:    make(map[string]map[string]bool),
	}
}

func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.strings[key]
	if ok {
		if data, ok := s.expiry[key]; ok {
			ttl := int(time.Until(data).Truncate(time.Second).Seconds())
			if ttl <= 0 {
				return false
			}
		}
	}
	return ok
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.strings[key]; ok {
		delete(s.strings, key)
		delete(s.expiry, key)
		return true
	}
	return false
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.strings[key]
	if ok {
		if data, ok := s.expiry[key]; ok {
			ttl := int(time.Until(data).Truncate(time.Second).Seconds())
			if ttl <= 0 {
				return "", false
			}
		}
		return data, true
	}
	return "", false
}

func (s *Store) SetWithExpiry(key string, value string, seconds time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strings[key] = value
	s.expiry[key] = seconds
}

func (s *Store) GetTTL(key string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if data, ok := s.expiry[key]; ok {
		return data, true
	}
	return time.Time{}, false
}

func (s *Store) SetExpire(key string, seconds time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.strings[key]; ok {
		s.expiry[key] = seconds
		return true
	}
	return false
}

func (s *Store) Set(key string, value string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strings == nil {
		return false
	}

	s.strings[key] = value
	return true
}

// Clean data that is expired
func (s *Store) CleanUp() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for key, exp := range s.expiry {
			if !exp.After(now) {
				delete(s.expiry, key)
				delete(s.strings, key)
			}
		}
		s.mu.Unlock()
	}
}

// LIST Fucntions

func (s *Store) LPush(key string, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.lists[key]
	data = append([]string{value}, data...)
	s.lists[key] = data
	return len(data)
}

func (s *Store) RPush(key string, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.lists[key]
	data = append(data, value)
	s.lists[key] = data
	return len(data)
}

func (s *Store) LPop(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.lists) <= 0 {
		return "", false
	}
	if data, ok := s.lists[key]; ok {
		response := data[0]
		s.lists[key] = data[1:]
		return response, true
	}

	return "", false
}

func (s *Store) RPop(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.lists) <= 0 {
		return "", false
	}
	if data, ok := s.lists[key]; ok {
		response := data[len(data)-1]
		s.lists[key] = data[:len(data)-1]
		return response, true
	}

	return "", false
}

func (s *Store) LRange(key string, start int, stop int) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if start < 0 || stop > len(s.lists[key]) {
		return nil, false
	}
	if start == 0 && stop <= -1 {
		return s.lists[key], true
	}
	response := make(map[string][]string)
	i := 0
	if _, ok := s.lists[key]; !ok {
		return nil, false
	}
	for _, value := range s.lists[key] {
		if i >= start && i <= stop {
			response[key] = append(response[key], value)
		}
		i++
	}
	return response[key], true
}

func (s *Store) LLen(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if data, ok := s.lists[key]; ok {
		return len(data)
	}
	return 0
}

// Hash Functions
func (s *Store) HSet(key string, field string, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ensure inner map exists
	if _, ok := s.hashes[key]; !ok {
		s.hashes[key] = make(map[string]string)
	}

	if _, existed := s.hashes[key][field]; existed {
		s.hashes[key][field] = value
		return 0
	}
	s.hashes[key][field] = value
	return 1
}

func (s *Store) HGet(key string, field string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.hashes[key]; ok {
		if response, ok2 := m[field]; ok2 {
			return response, true
		}
	}
	return "", false
}

func (s *Store) HDel(key string, field string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.hashes[key]; ok {
		if _, ok2 := m[field]; ok2 {
			delete(m, field)
			return 1
		}
	}
	return 0
}

func (s *Store) HGetAll(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	response := make([]string, 0)
	if m, ok := s.hashes[key]; ok {
		for k, v := range m {
			response = append(response, k)
			response = append(response, v)
		}
	}
	return response
}

func (s *Store) HKeys(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	response := make([]string, 0)
	if m, ok := s.hashes[key]; ok {
		for field := range m {
			response = append(response, field)
		}
	}
	return response
}

func (s *Store) HLen(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.hashes[key]
	if !ok {
		return 0
	}
	return len(s.hashes[key])
}

// SETS (unordered) Functions

func (s *Store) SAdd(key string, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	// ensure inner map exists
	if _, ok := s.sets[key]; !ok {
		s.sets[key] = make(map[string]bool)
	}
	if _, ok := s.sets[key]; ok {
		if _, ok := s.sets[key][value]; ok {
			s.sets[key][value] = true
			return 0
		}
		s.sets[key][value] = true
		return 1
	}
	s.sets[key][value] = true
	return 1
}

func (s *Store) SRem(key string, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sets[key][value]; ok {
		delete(s.sets[key], value)
		return 1
	}
	return 0
}

func (s *Store) SMembers(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	response := make([]string, 0)
	for value, ok := range s.sets[key] {
		if ok {
			response = append(response, value)
		}
	}
	return response
}

func (s *Store) SIsMember(key string, member string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.sets[key][member]; ok {
		return 1
	}
	return 0
}

func (s *Store) SCard(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	response := 0

	for _, ok := range s.sets[key] {
		if ok {
			response++
		}
	}
	return response
}

// Snapshot is an immutable, deep-copied view of every key space. The
// persistence layer serialises this value instead of touching the live maps,
// so a slow disk write never blocks running commands.
type Snapshot struct {
	Strings map[string]string
	Expiry  map[string]time.Time
	Lists   map[string][]string
	Hashes  map[string]map[string]string
	Sets    map[string]map[string]bool
}

// SnapShot (to avoid slow write operation and save data later, user won't be stopped)
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Strings
	redisData := make(map[string]string)
	for k, v := range s.strings {
		redisData[k] = v
	}

	// Expiry
	expiry := make(map[string]time.Time)
	for k, v := range s.expiry {
		expiry[k] = v
	}

	// Lists
	lists := make(map[string][]string)
	for k, v := range s.lists {
		lists[k] = v
	}

	// Hash
	hashes := make(map[string]map[string]string)
	for k, v := range s.hashes {
		hashes[k] = make(map[string]string)
		for k1, v1 := range v {
			hashes[k][k1] = v1
		}
	}

	// Sets
	sets := make(map[string]map[string]bool)
	for k, v := range s.sets {
		sets[k] = make(map[string]bool)
		for k1, v1 := range v {
			sets[k][k1] = v1
		}
	}

	return Snapshot{
		Strings: redisData,
		Expiry:  expiry,
		Lists:   lists,
		Hashes:  hashes,
		Sets:    sets,
	}
}

// Restore loads a previously saved Snapshot back into the live store. Expired
// string keys are dropped on the way in so a stale dump never resurrects data.
func (s *Store) Restore(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TTL
	for k, v := range snap.Expiry {
		s.expiry[k] = v
	}

	// Strings( Redis_data )
	for k, v := range snap.Strings {
		if duration, ok := s.expiry[k]; ok {
			if int(time.Until(duration).Truncate(time.Second).Seconds()) < 0 {
				delete(s.expiry, k)
				continue
			}
		}
		s.strings[k] = v
	}

	// Lists
	s.lists = snap.Lists

	// SETS
	for k, v := range snap.Sets {
		s.sets[k] = v
	}

	// HASHES
	for k, v := range snap.Hashes {
		s.hashes[k] = v
	}
}
