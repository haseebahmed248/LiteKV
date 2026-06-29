package main

import (
	"flag"
	"log"

	"litekv/internal/commands"
	"litekv/internal/persistence"
	"litekv/internal/pubsub"
	"litekv/internal/server"
	"litekv/internal/store"
)

// main is the composition root: it parses configuration, constructs every
// component once, wires them together via dependency injection, and starts the
// server. Nothing else in the program reaches for global state.
func main() {
	addr := flag.String("addr", "localhost:6379", "address the server listens on")
	dataPath := flag.String("data", "data.json", "path to the persistence dump file")
	flag.Parse()

	st := store.New()
	broker := pubsub.New()
	pm := persistence.New(st, *dataPath)
	router := commands.New(st, broker, pm)

	srv := server.New(*addr, st, router, broker, pm)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
