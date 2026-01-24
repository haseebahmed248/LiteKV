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

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	subscribed := false
	for {
		args, err := protocol.Parse(reader)

		if err != nil {
			log.Print(err)
			return
		}

		if subscribed {
			if args[0] != "SUBSCRIBE" && args[0] != "UNSUBSCRIBE" && args[0] != "PING" {
				conn.Write([]byte(protocol.SerializeError("only SUBSCRIBE/UNSUBSCRIBE/PING allowed")))
				continue
			}
		}
		if args[0] == "SUBSCRIBE" {
			subscribed = true
		}

		response, err := commands.Route(args, conn)
		if err != nil || response == "" {
			log.Print(err)
			log.Print(response)
		}
		if args[0] == "UNSUBSCRIBE" {
			if !pubsub.IsSubscribed(conn) {
				subscribed = false
			}
		}
		conn.Write([]byte(response))
	}

}

func StartServer() {
	connection, err := net.Listen("tcp", "localhost:6379")
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Listening to port 6379")
	go store.CleanUp()
	persistence.Load()
	for {
		conn, err := connection.Accept()
		if err != nil {
			log.Print(err)
			return
		}

		go handleConnection(conn)
	}
}
