package commands

import (
	"errors"
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
	} else {
		return protocol.SerializeError("Invalid operation"), errors.New("invalid Operation")
	}
}
