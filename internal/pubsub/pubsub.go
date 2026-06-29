// Package pubsub implements channel-based fan-out messaging. A Broker tracks
// which connections subscribe to which channels and pushes published messages
// to every subscriber. State is held on the Broker (not in globals) so it can
// be injected wherever it is needed.
package pubsub

import (
	"litekv/internal/protocol"
	"net"
	"slices"
	"sync"
)

// Broker maps each channel name to the connections subscribed to it.
type Broker struct {
	mu       sync.Mutex
	channels map[string][]net.Conn
}

// New builds an empty Broker ready to accept subscriptions.
func New() *Broker {
	return &Broker{
		channels: make(map[string][]net.Conn),
	}
}

func (b *Broker) Subscribe(channel string, conn net.Conn) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	// need to check and remove duplicates
	if slices.Contains(b.channels[channel], conn) {
		return len(b.channels[channel])
	} else {
		b.channels[channel] = append(b.channels[channel], conn)
		return len(b.channels[channel])
	}
}

func (b *Broker) Unsubscribe(channel string, conn net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	conns := b.channels[channel]
	for i, c := range conns {
		if c == conn {
			b.channels[channel] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
}

func (b *Broker) Publish(channel string, message string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := []string{"message", channel, message}
	total := 0
	response := protocol.SerializeArray(data)
	if v, ok := b.channels[channel]; ok {
		for _, v1 := range v {
			v1.Write([]byte(response))
			total++
		}
	}
	return total
}

func (b *Broker) IsSubscribed(conn net.Conn) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, v := range b.channels {
		if slices.Contains(v, conn) {
			return true
		}
	}
	return false
}
