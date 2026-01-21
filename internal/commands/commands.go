package commands

import (
	"errors"
	"strconv"
	"time"

	"litekv/internal/protocol"
	"litekv/internal/store"
)

func Route(parsed []string) (string, error) {
	if string(parsed[0]) == "PING" {
		return protocol.SerializeSimpleString("PONG"), nil
	}
	if string(parsed[0]) == "GET" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("GET requires a key")
		}
		data, ok := store.Get(parsed[1])
		if ok {
			response := protocol.SerializeBulkString(data)
			return response, nil
		}
		return protocol.SerializeNull(), errors.New("Value doesn't exist")
	} else if string(parsed[0]) == "SET" {
		if len(parsed) <= 2 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("Error Setting data")
		}
		if store.Set(string(parsed[1]), string(parsed[2])) {
			return protocol.SerializeSimpleString("OK"), nil
		}
		return protocol.SerializeError("Internal Error"), errors.New("Error Setting data")
	} else if string(parsed[0]) == "DEL" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("DEL requires a key")
		}

		if store.Delete(parsed[1]) {
			return protocol.SerializeInteger(1), nil
		}
		return protocol.SerializeInteger(0), nil
	} else if string(parsed[0]) == "EXISTS" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("EXISTS requires a key")
		}
		if store.Exists(parsed[1]) {
			return protocol.SerializeInteger(1), nil
		}
		return protocol.SerializeInteger(0), nil
	} else if string(parsed[0]) == "SETEX" {
		if len(parsed) < 4 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("SETEX requires keys")
		}
		seconds, err := strconv.Atoi(parsed[2])
		if err != nil {
			return protocol.SerializeError("Value is not an integer or out of range"), errors.New("invalid expiry")
		}
		expiry := time.Now().Add(time.Duration(seconds) * time.Second)
		store.SetWithExpiry(string(parsed[1]), string(parsed[3]), expiry)
		return protocol.SerializeSimpleString("OK"), nil
	} else if string(parsed[0]) == "TTL" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("TTL requires more arguments")
		}
		data, ok := store.GetTTL(parsed[1])
		if ok {
			ttl := int(time.Until(data).Truncate(time.Second).Seconds())
			if store.Exists(parsed[1]) {
				return protocol.SerializeInteger(-1), nil
			}
			if ttl <= 0 {
				return protocol.SerializeInteger(-2), nil
			}
			return protocol.SerializeInteger(ttl), nil
		}
		return protocol.SerializeInteger(-2), errors.New("TTL fetch error")
	} else if string(parsed[0]) == "EXPIRE" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments"), errors.New("EXPIRE requires more arguments")
		}
		seconds, err := strconv.Atoi(parsed[2])
		if err != nil {
			return protocol.SerializeError("Value is not an integer or out of range"), errors.New("invalid expiry")
		}
		expiry := time.Now().Add(time.Duration(seconds) * time.Second)
		if store.SetExpire(parsed[1], expiry) {
			return protocol.SerializeInteger(1), nil
		}
		return protocol.SerializeInteger(0), errors.New("Key doesn't exists")
	} else {
		return protocol.SerializeError("Invalid operation"), errors.New("invalid Operation")
	}
}
