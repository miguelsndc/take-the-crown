package main

import (
	"log"
	"github.com/miguelsndc/multiplayer-server/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal("Error while setting up the server")
	}
}
