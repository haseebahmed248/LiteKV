// Package commands turns a parsed RESP request into a response. It uses the
// Command pattern: every command is a Handler registered in a dispatch table,
// so adding a command means adding one map entry instead of growing an
// if/else chain. The Router holds the dependencies (store, pubsub, persistence)
// that handlers need, injected once at construction.
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

// Handler executes a single command. Every handler shares this signature so the
// dispatch table can treat them uniformly; handlers that don't need the
// connection simply ignore it.
type Handler func(parsed []string, conn net.Conn) (string, error)

// Router dispatches a parsed command to its Handler.
type Router struct {
	store       *store.Store
	pubsub      *pubsub.Broker
	persistence *persistence.Manager
	handlers    map[string]Handler
}

// New builds a Router with its dependencies and registers the command table.
func New(s *store.Store, b *pubsub.Broker, p *persistence.Manager) *Router {
	r := &Router{store: s, pubsub: b, persistence: p}
	r.handlers = map[string]Handler{
		"PING":        r.ping,
		"GET":         r.get,
		"SET":         r.set,
		"DEL":         r.del,
		"EXISTS":      r.exists,
		"SETEX":       r.setex,
		"TTL":         r.ttl,
		"EXPIRE":      r.expire,
		"LPUSH":       r.lpush,
		"RPUSH":       r.rpush,
		"LPOP":        r.lpop,
		"RPOP":        r.rpop,
		"LRANGE":      r.lrange,
		"LLEN":        r.llen,
		"HSET":        r.hset,
		"HGET":        r.hget,
		"HLEN":        r.hlen,
		"HGETALL":     r.hgetall,
		"HDEL":        r.hdel,
		"HKEYS":       r.hkeys,
		"SADD":        r.sadd,
		"SREM":        r.srem,
		"SISMEMBER":   r.sismember,
		"SCARD":       r.scard,
		"SMEMBERS":    r.smembers,
		"SAVE":        r.save,
		"BGSAVE":      r.bgsave,
		"SUBSCRIBE":   r.subscribe,
		"UNSUBSCRIBE": r.unsubscribe,
		"PUBLISH":     r.publish,
	}
	return r
}

// Route looks up the command in the dispatch table and runs it.
func (r *Router) Route(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) == 0 {
		return protocol.SerializeError("empty command"), errors.New("empty command")
	}
	handler, ok := r.handlers[parsed[0]]
	if !ok {
		return protocol.SerializeError("Invalid operation"), errors.New("invalid Operation")
	}
	return handler(parsed, conn)
}

func (r *Router) ping(parsed []string, conn net.Conn) (string, error) {
	return protocol.SerializeSimpleString("PONG"), nil
}

func (r *Router) get(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("GET requires a key")
	}
	data, ok := r.store.Get(parsed[1])
	if ok {
		response := protocol.SerializeBulkString(data)
		return response, nil
	}
	return protocol.SerializeNull(), errors.New("Value doesn't exist")
}

func (r *Router) set(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) <= 2 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("Error Setting data")
	}
	if r.store.Set(string(parsed[1]), string(parsed[2])) {
		return protocol.SerializeSimpleString("OK"), nil
	}
	return protocol.SerializeError("Internal Error"), errors.New("Error Setting data")
}

func (r *Router) del(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("DEL requires a key")
	}

	if r.store.Delete(parsed[1]) {
		return protocol.SerializeInteger(1), nil
	}
	return protocol.SerializeInteger(0), nil
}

func (r *Router) exists(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("EXISTS requires a key")
	}
	if r.store.Exists(parsed[1]) {
		return protocol.SerializeInteger(1), nil
	}
	return protocol.SerializeInteger(0), nil
}

func (r *Router) setex(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 4 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("SETEX requires keys")
	}
	seconds, err := strconv.Atoi(parsed[2])
	if err != nil {
		return protocol.SerializeError("Value is not an integer or out of range"), errors.New("invalid expiry")
	}
	expiry := time.Now().Add(time.Duration(seconds) * time.Second)
	r.store.SetWithExpiry(string(parsed[1]), string(parsed[3]), expiry)
	return protocol.SerializeSimpleString("OK"), nil
}

func (r *Router) ttl(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("TTL requires more arguments")
	}
	data, ok := r.store.GetTTL(parsed[1])
	if ok {
		ttl := int(time.Until(data).Truncate(time.Second).Seconds())
		if r.store.Exists(parsed[1]) {
			return protocol.SerializeInteger(-1), nil
		}
		if ttl <= 0 {
			return protocol.SerializeInteger(-2), nil
		}
		return protocol.SerializeInteger(ttl), nil
	}
	return protocol.SerializeInteger(-2), errors.New("TTL fetch error")
}

func (r *Router) expire(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments"), errors.New("EXPIRE requires more arguments")
	}
	seconds, err := strconv.Atoi(parsed[2])
	if err != nil {
		return protocol.SerializeError("Value is not an integer or out of range"), errors.New("invalid expiry")
	}
	expiry := time.Now().Add(time.Duration(seconds) * time.Second)
	if r.store.SetExpire(parsed[1], expiry) {
		return protocol.SerializeInteger(1), nil
	}
	return protocol.SerializeInteger(0), errors.New("Key doesn't exists")
}

func (r *Router) lpush(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'LPUSH' command"), errors.New("Wrong number of arguments for LPUSH command")
	}
	response := r.store.LPush(parsed[1], parsed[2])
	return protocol.SerializeInteger(response), nil
}

func (r *Router) rpush(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'RPUSH' command"), errors.New("Wrong number of arguments for RPUSH command")
	}
	response := r.store.RPush(parsed[1], parsed[2])
	return protocol.SerializeInteger(response), nil
}

func (r *Router) lpop(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 1 {
		return protocol.SerializeError("Wrong number of arguments for 'LPOP' command"), errors.New("Wrong number of arguments for LPOP command")
	}
	response, ok := r.store.LPop(parsed[1])
	if ok {
		return protocol.SerializeBulkString(response), nil
	}
	return protocol.SerializeNull(), errors.New("Error LPOP data")
}

func (r *Router) rpop(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 1 {
		return protocol.SerializeError("Wrong number of arguments for 'LPOP' command"), errors.New("Wrong number of arguments for LPOP command")
	}
	response, ok := r.store.RPop(parsed[1])
	if ok {
		return protocol.SerializeBulkString(response), nil
	}
	return protocol.SerializeNull(), errors.New("Error LPOP data")
}

func (r *Router) lrange(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'LRANGE' command"), errors.New("Wrong number of arguments for LRANGE command")
	}
	start, _ := strconv.Atoi(parsed[2])
	end, _ := strconv.Atoi(parsed[3])
	response, ok := r.store.LRange(parsed[1], start, end)
	if ok {
		return protocol.SerializeArray(response), nil
	}
	return protocol.SerializeNull(), nil
}

func (r *Router) llen(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 1 {
		return protocol.SerializeError("Wrong number of arguments for 'LLEN' command"), errors.New("Wrong number of arguments for 'LLEN' command")
	}
	response := r.store.LLen(parsed[1])
	return protocol.SerializeInteger(response), nil
}

func (r *Router) hset(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 4 {
		return protocol.SerializeError("Wrong number of arguments for 'HSET' command"), errors.New("Wrong number of arguments for 'HSET' command")
	}
	return protocol.SerializeInteger(r.store.HSet(parsed[1], parsed[2], parsed[3])), nil
}

func (r *Router) hget(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'HGET' command"), errors.New("Wrong number of arguments for 'HGET' command")
	}
	if response, ok := r.store.HGet(parsed[1], parsed[2]); ok {
		return protocol.SerializeSimpleString(response), nil
	}
	return protocol.SerializeNull(), errors.New("No data found")
}

func (r *Router) hlen(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'HLEN' command"), errors.New("Wrong number of arguments for 'HLEN' command")
	}
	return protocol.SerializeInteger(r.store.HLen(parsed[1])), nil
}

func (r *Router) hgetall(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'HGETALL' command"), errors.New("Wrong number of arguments for 'HGETALL' command")
	}
	response := r.store.HGetAll(parsed[1])
	return protocol.SerializeArray(response), nil
}

func (r *Router) hdel(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'HDEL' command"), errors.New("Wrong number of arguments for 'HDEL' command")
	}
	return protocol.SerializeInteger(r.store.HDel(parsed[1], parsed[2])), nil
}

func (r *Router) hkeys(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'HDEL' command"), errors.New("Wrong number of arguments for 'HDEL' command")
	}
	response := r.store.HKeys(parsed[1])
	return protocol.SerializeArray(response), nil
}

func (r *Router) sadd(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'SADD' command"), errors.New("Wrong number of arguments for 'SADD' command")
	}
	return protocol.SerializeInteger(r.store.SAdd(parsed[1], parsed[2])), nil
}

func (r *Router) srem(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'SREM' command"), errors.New("Wrong number of arguments for 'SREM' command")
	}
	return protocol.SerializeInteger(r.store.SRem(parsed[1], parsed[2])), nil
}

func (r *Router) sismember(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'SISMEMBER' command"), errors.New("Wrong number of arguments for 'SISMEMBER' command")
	}
	return protocol.SerializeInteger(r.store.SIsMember(parsed[1], parsed[2])), nil
}

func (r *Router) scard(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'SCARD' command"), errors.New("Wrong number of arguments for 'SCARD' command")
	}
	return protocol.SerializeInteger(r.store.SCard(parsed[1])), nil
}

func (r *Router) smembers(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'SMEMBERS' command"), errors.New("Wrong number of arguments for 'SMEMBERS' command")
	}
	return protocol.SerializeArray(r.store.SMembers(parsed[1])), nil
}

func (r *Router) save(parsed []string, conn net.Conn) (string, error) {
	if r.persistence.Save() {
		return protocol.SerializeSimpleString("OK"), nil
	}
	return protocol.SerializeError("Error saving data"), nil
}

func (r *Router) bgsave(parsed []string, conn net.Conn) (string, error) {
	go r.persistence.Save()
	return protocol.SerializeSimpleString("Background saving started"), nil
}

func (r *Router) subscribe(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'SUBSCRIBE' command"), errors.New("Wrong number of arguments for 'SUBSCRIBE' command")
	}
	response := r.pubsub.Subscribe(parsed[1], conn)
	return protocol.SerializeArray([]string{"subscribe", parsed[1], strconv.Itoa(response)}), nil
}

func (r *Router) unsubscribe(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 2 {
		return protocol.SerializeError("Wrong number of arguments for 'UNSUBSCRIBE' command"), errors.New("Wrong number of arguments for 'UNSUBSCRIBE' command")
	}
	r.pubsub.Unsubscribe(parsed[1], conn)
	return protocol.SerializeSimpleString("Unsubscribed successfully"), nil
}

func (r *Router) publish(parsed []string, conn net.Conn) (string, error) {
	if len(parsed) < 3 {
		return protocol.SerializeError("Wrong number of arguments for 'PUBLISH' command"), errors.New("Wrong number of arguments for 'PUBLISH' command")
	}
	response := r.pubsub.Publish(parsed[1], parsed[2])
	return protocol.SerializeInteger(response), nil
}
