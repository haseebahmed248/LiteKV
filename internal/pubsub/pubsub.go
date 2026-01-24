package pubsub

import (
	"litekv/internal/protocol"
	"net"
	"slices"
	"sync"
)

var channels = make(map[string][]net.Conn)
var mu sync.Mutex

func Subscribe(channel string, conn net.Conn) int {
	mu.Lock()
	defer mu.Unlock()
	// need to check and remove duplicates
	if slices.Contains(channels[channel], conn) {
		return len(channels[channel])
	} else {
		channels[channel] = append(channels[channel], conn)
		return len(channels[channel])
	}
}

func Unsubscribe(channel string, conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	conns := channels[channel]
	for i, c := range conns {
		if c == conn {
			channels[channel] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
}

func Publish(channel string, message string) int {
	mu.Lock()
	defer mu.Unlock()
	data := []string{"message", channel, message}
	total := 0
	response := protocol.SerializeArray(data)
	if v, ok := channels[channel]; ok {
		for _, v1 := range v {
			v1.Write([]byte(response))
			total++
		}
	}
	return total
}

func IsSubscribed(conn net.Conn) bool {
	mu.Lock()
	defer mu.Unlock()
	for _, v := range channels {
		if slices.Contains(v, conn) {
			return true
		}
	}
	return false
}
