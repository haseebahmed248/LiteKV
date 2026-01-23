package persistence

import (
	"encoding/json"
	"litekv/internal/store"
	"log"
	"os"
	"time"
)

type Database struct {
	Strings map[string]string            `json:"strings"`
	Lists   map[string][]string          `json:"lists"`
	Hashes  map[string]map[string]string `json:"hashes"`
	Sets    map[string][]string          `json:"sets"`
	Expiry  map[string]time.Time         `json:"expiry"`
}

func Save() bool {
	var config Database
	config.Sets = make(map[string][]string)
	config.Hashes = map[string]map[string]string{}
	config.Expiry = map[string]time.Time{}

	strings, expiry, lists, hashes, sets := store.GetSnapshot()
	config.Strings = strings
	config.Expiry = expiry
	config.Lists = lists
	config.Hashes = hashes

	// Convert sets from map[string]bool to []string
	for k, v := range sets {
		for item, exists := range v {
			if exists {
				config.Sets[k] = append(config.Sets[k], item)
			}
		}
	}

	file, err := os.OpenFile(
		"data.json",
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

func Load() {
	file, err := os.OpenFile(
		"data.json",
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
	// TTL
	for k, v := range data.Expiry {
		store.Expiry[k] = v
	}

	// Strings( Redis_data )
	for k, v := range data.Strings {
		if duration, ok := store.Expiry[k]; ok {
			if int(time.Until(duration).Truncate(time.Second).Seconds()) < 0 {
				delete(store.Expiry, k)
				continue
			}
		}
		store.Redis_data[k] = v
	}

	// Lists
	store.List_data = data.Lists

	// SETS
	for k, v := range data.Sets {
		setMap := make(map[string]bool)
		for _, item := range v {
			setMap[item] = true
		}
		store.Set_data[k] = setMap
	}

	// HASHES
	for k, v := range data.Hashes {
		store.Hash_data[k] = v
	}

}
