// Package persistence snapshots the in-memory store to disk as JSON and loads
// it back on startup. A Manager owns the target path and a reference to the
// Store it serialises, so saving/loading is a method call rather than a reach
// into package globals.
package persistence

import (
	"encoding/json"
	"litekv/internal/store"
	"log"
	"os"
	"time"
)

// Database is the on-disk shape of the data. Sets are stored as []string (JSON
// has no set type) and converted to/from the store's map representation.
type Database struct {
	Strings map[string]string            `json:"strings"`
	Lists   map[string][]string          `json:"lists"`
	Hashes  map[string]map[string]string `json:"hashes"`
	Sets    map[string][]string          `json:"sets"`
	Expiry  map[string]time.Time         `json:"expiry"`
}

// Manager coordinates persistence for a single Store and dump file.
type Manager struct {
	store *store.Store
	path  string
}

// New wires a Manager to the store it persists and the file it writes to.
func New(s *store.Store, path string) *Manager {
	return &Manager{store: s, path: path}
}

func (m *Manager) Save() bool {
	var config Database
	config.Sets = make(map[string][]string)
	config.Hashes = map[string]map[string]string{}
	config.Expiry = map[string]time.Time{}

	snap := m.store.Snapshot()
	config.Strings = snap.Strings
	config.Expiry = snap.Expiry
	config.Lists = snap.Lists
	config.Hashes = snap.Hashes

	// Convert sets from map[string]bool to []string
	for k, v := range snap.Sets {
		for item, exists := range v {
			if exists {
				config.Sets[k] = append(config.Sets[k], item)
			}
		}
	}

	file, err := os.OpenFile(
		m.path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		log.Print("Error saving Data, ", err)
		return false
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(&config); err != nil {
		log.Print("Error saving data in file: ", err)
		return false
	}
	return true

}

func (m *Manager) Load() {
	file, err := os.OpenFile(
		m.path,
		os.O_RDWR|os.O_CREATE,
		0644,
	)
	if err != nil {
		log.Print(err)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if info.Size() == 0 {
		log.Print("Database is empty. Starting from fresh")
		return
	}

	decoder := json.NewDecoder(file)
	var data Database
	if err := decoder.Decode(&data); err != nil {
		log.Printf("Error decoding json file, %v", err)
		return
	}

	// Convert sets from []string back to map[string]bool for the live store.
	sets := make(map[string]map[string]bool)
	for k, v := range data.Sets {
		setMap := make(map[string]bool)
		for _, item := range v {
			setMap[item] = true
		}
		sets[k] = setMap
	}

	m.store.Restore(store.Snapshot{
		Strings: data.Strings,
		Expiry:  data.Expiry,
		Lists:   data.Lists,
		Hashes:  data.Hashes,
		Sets:    sets,
	})
}
