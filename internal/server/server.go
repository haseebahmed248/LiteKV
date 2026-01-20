package server

import (
	"bufio"
	"litekv/internal/protocol"
	"log"
	"net"
)

func handleConnection(conn net.Conn) {
	for {
		reader := bufio.NewReader(conn)
		args, err := protocol.Parse(reader)
		if err != nil {
			log.Print(err)
			return
		}
		defer conn.Close()
		if args[0] == "PING" {
			response := protocol.SerializeSimpleString("PONG")
			conn.Write([]byte(response))
		}
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
