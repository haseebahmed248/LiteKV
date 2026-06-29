// Package server owns the TCP front door. A Server accepts connections and,
// for each one, parses RESP requests and hands them to the command Router. All
// collaborators (store, router, pubsub, persistence) are injected at
// construction so the server has no hidden global dependencies.
package server

import (
	"bufio"
	"litekv/internal/commands"
	"litekv/internal/persistence"
	"litekv/internal/protocol"
	"litekv/internal/pubsub"
	"litekv/internal/store"
	"log"
	"net"
)

// Server ties together everything needed to serve clients on a TCP address.
type Server struct {
	addr        string
	store       *store.Store
	router      *commands.Router
	pubsub      *pubsub.Broker
	persistence *persistence.Manager
}

// New constructs a Server listening on addr with its dependencies injected.
func New(addr string, s *store.Store, router *commands.Router, b *pubsub.Broker, p *persistence.Manager) *Server {
	return &Server{
		addr:        addr,
		store:       s,
		router:      router,
		pubsub:      b,
		persistence: p,
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	subscribed := false
	for {
		args, err := protocol.Parse(reader)

		if err != nil {
			log.Print(err)
			return
		}

		if subscribed {
			if args[0] != "SUBSCRIBE" && args[0] != "UNSUBSCRIBE" && args[0] != "PING" {
				writer.Write([]byte(protocol.SerializeError("only SUBSCRIBE/UNSUBSCRIBE/PING allowed")))
				if reader.Buffered() == 0 {
					writer.Flush()
				}
				continue
			}
		}
		if args[0] == "SUBSCRIBE" {
			subscribed = true
		}

		response, err := s.router.Route(args, conn)
		if err != nil || response == "" {
			log.Print(err)
			log.Print(response)
		}
		if args[0] == "UNSUBSCRIBE" {
			if !s.pubsub.IsSubscribed(conn) {
				subscribed = false
			}
		}
		writer.Write([]byte(response))
		if reader.Buffered() == 0 {
			writer.Flush()
		}
	}

}

// Start loads any persisted data, launches the background expiry sweeper, and
// blocks accepting connections until the listener fails.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Print(err)
		return err
	}
	log.Printf("Listening to %s", s.addr)
	go s.store.CleanUp()
	s.persistence.Load()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			return err
		}

		go s.handleConnection(conn)
	}
}
