package commands

import (
	"errors"
	"litekv/internal/store"
)

func Route(parsed []string) (string, error) {
	if string(parsed[0]) == "PING" {
		return "PONG", nil
	}
	if string(parsed[0]) == "GET" {
		if len(parsed) < 2 {
			return "", errors.New("GET requires a key")
		}
		data, ok := store.Get(parsed[1])
		if ok {
			return data, nil
		}
		return "", errors.New("Value doesn't exsist")
	} else if string(parsed[0]) == "SET" {
		if len(parsed) <= 2 {
			return "", errors.New("Error Setting data")
		}
		if store.Set(string(parsed[1]), string(parsed[2])) {
			return "OK", nil
		}
		return "", errors.New("Error Setting data")
	} else {
		return "", errors.New("invalid Operation")
	}
}
