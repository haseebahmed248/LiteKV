package commands

import (
	"errors"
	"net"
	"strconv"
	"time"

	"litekv/internal/persistence"
	"litekv/internal/protocol"
	"litekv/internal/pubsub"
	"litekv/internal/store"
)

func Route(parsed []string, conn net.Conn) (string, error) {
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
	} else if string(parsed[0]) == "LPUSH" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'LPUSH' command"), errors.New("Wrong number of arguments for LPUSH command")
		}
		response := store.LPush(parsed[1], parsed[2])
		return protocol.SerializeInteger(response), nil

	} else if string(parsed[0]) == "RPUSH" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'RPUSH' command"), errors.New("Wrong number of arguments for RPUSH command")
		}
		response := store.RPush(parsed[1], parsed[2])
		return protocol.SerializeInteger(response), nil
	} else if string(parsed[0]) == "LPOP" {
		if len(parsed) < 1 {
			return protocol.SerializeError("Wrong number of arguments for 'LPOP' command"), errors.New("Wrong number of arguments for LPOP command")
		}
		response, ok := store.LPop(parsed[1])
		if ok {
			return protocol.SerializeBulkString(response), nil
		}
		return protocol.SerializeNull(), errors.New("Error LPOP data")
	} else if string(parsed[0]) == "RPOP" {
		if len(parsed) < 1 {
			return protocol.SerializeError("Wrong number of arguments for 'LPOP' command"), errors.New("Wrong number of arguments for LPOP command")
		}
		response, ok := store.RPop(parsed[1])
		if ok {
			return protocol.SerializeBulkString(response), nil
		}
		return protocol.SerializeNull(), errors.New("Error LPOP data")
	} else if string(parsed[0]) == "LRANGE" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'LRANGE' command"), errors.New("Wrong number of arguments for LRANGE command")
		}
		start, _ := strconv.Atoi(parsed[2])
		end, _ := strconv.Atoi(parsed[3])
		response, ok := store.LRange(parsed[1], start, end)
		if ok {
			return protocol.SerializeArray(response), nil
		}
		return protocol.SerializeNull(), nil
	} else if string(parsed[0]) == "LLEN" {
		if len(parsed) < 1 {
			return protocol.SerializeError("Wrong number of arguments for 'LLEN' command"), errors.New("Wrong number of arguments for 'LLEN' command")
		}
		response := store.LLen(parsed[1])
		return protocol.SerializeInteger(response), nil
	} else if string(parsed[0]) == "HSET" {
		if len(parsed) < 4 {
			return protocol.SerializeError("Wrong number of arguments for 'HSET' command"), errors.New("Wrong number of arguments for 'HSET' command")
		}
		return protocol.SerializeInteger(store.HSet(parsed[1], parsed[2], parsed[3])), nil

	} else if string(parsed[0]) == "HGET" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'HGET' command"), errors.New("Wrong number of arguments for 'HGET' command")
		}
		if response, ok := store.HGet(parsed[1], parsed[2]); ok {
			return protocol.SerializeSimpleString(response), nil
		}
		return protocol.SerializeNull(), errors.New("No data found")
	} else if string(parsed[0]) == "HLEN" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'HLEN' command"), errors.New("Wrong number of arguments for 'HLEN' command")
		}
		return protocol.SerializeInteger(store.HLen(parsed[1])), nil
	} else if string(parsed[0]) == "HGETALL" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'HGETALL' command"), errors.New("Wrong number of arguments for 'HGETALL' command")
		}
		response := store.HGetAll(parsed[1])
		return protocol.SerializeArray(response), nil
	} else if string(parsed[0]) == "HDEL" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'HDEL' command"), errors.New("Wrong number of arguments for 'HDEL' command")
		}
		return protocol.SerializeInteger(store.HDel(parsed[1], parsed[2])), nil
	} else if string(parsed[0]) == "HKEYS" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'HDEL' command"), errors.New("Wrong number of arguments for 'HDEL' command")
		}
		response := store.HKeys(parsed[1])
		return protocol.SerializeArray(response), nil
	} else if string(parsed[0]) == "SADD" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'SADD' command"), errors.New("Wrong number of arguments for 'SADD' command")
		}
		return protocol.SerializeInteger(store.SAdd(parsed[1], parsed[2])), nil

	} else if string(parsed[0]) == "SREM" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'SREM' command"), errors.New("Wrong number of arguments for 'SREM' command")
		}
		return protocol.SerializeInteger(store.SRem(parsed[1], parsed[2])), nil
	} else if string(parsed[0]) == "SISMEMBER" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'SISMEMBER' command"), errors.New("Wrong number of arguments for 'SISMEMBER' command")
		}
		return protocol.SerializeInteger(store.SIsMember(parsed[1], parsed[2])), nil
	} else if string(parsed[0]) == "SCARD" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'SCARD' command"), errors.New("Wrong number of arguments for 'SCARD' command")
		}
		return protocol.SerializeInteger(store.SCard(parsed[1])), nil
	} else if string(parsed[0]) == "SMEMBERS" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'SMEMBERS' command"), errors.New("Wrong number of arguments for 'SMEMBERS' command")
		}
		return protocol.SerializeArray(store.SMembers(parsed[1])), nil
	} else if string(parsed[0]) == "SAVE" {
		if persistence.Save() {
			return protocol.SerializeSimpleString("OK"), nil
		}
		return protocol.SerializeError("Error saving data"), nil
	} else if string(parsed[0]) == "BGSAVE" {
		go persistence.Save()
		return protocol.SerializeSimpleString("Background saving started"), nil
	} else if string(parsed[0]) == "SUBSCRIBE" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'SUBSCRIBE' command"), errors.New("Wrong number of arguments for 'SUBSCRIBE' command")
		}
		response := pubsub.Subscribe(parsed[1], conn)
		return protocol.SerializeArray([]string{"subscribe", parsed[1], strconv.Itoa(response)}), nil

	} else if string(parsed[0]) == "UNSUBSCRIBE" {
		if len(parsed) < 2 {
			return protocol.SerializeError("Wrong number of arguments for 'UNSUBSCRIBE' command"), errors.New("Wrong number of arguments for 'UNSUBSCRIBE' command")
		}
		pubsub.Unsubscribe(parsed[1], conn)
		return protocol.SerializeSimpleString("Unsubscribed successfully"), nil
	} else if string(parsed[0]) == "PUBLISH" {
		if len(parsed) < 3 {
			return protocol.SerializeError("Wrong number of arguments for 'PUBLISH' command"), errors.New("Wrong number of arguments for 'PUBLISH' command")
		}
		response := pubsub.Publish(parsed[1], parsed[2])
		return protocol.SerializeInteger(response), nil
	} else {
		return protocol.SerializeError("Invalid operation"), errors.New("invalid Operation")
	}
}
