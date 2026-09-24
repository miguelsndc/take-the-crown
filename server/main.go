package server

import (
	"log"
	"net"
	"net/http"
	"strconv"
)

func Run() error {
	const (
		port = 3000
	)
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fileServer)
	actualPort := ":" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", actualPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Server listening at port %d", port)
	return http.Serve(listener, mux)
}
