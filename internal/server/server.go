package server

import (
	"bufio"
	"litekv/internal/commands"
	"litekv/internal/protocol"
	"log"
	"net"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		args, err := protocol.Parse(reader)
		if err != nil {
			log.Print(err)
			return
		}
		response, err := commands.Route(args)
		if err != nil || response == "" {
			log.Print(err)
			log.Print(response)
			// conn.Write([]byte(protocol.SerializeError("-1")))
			// continue
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
	for {
		conn, err := connection.Accept()
		if err != nil {
			log.Print(err)
			return
		}

		go handleConnection(conn)
	}
}
