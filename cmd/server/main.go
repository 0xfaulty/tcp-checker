package main

import (
	"flag"
	"log"
	"net"

	"github.com/0xfaulty/tcp-checker/internal/checker"
)

func main() {
	port := flag.String("port", "7300", "Port to listen on")
	flag.Parse()

	address := ":" + *port
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
	}()

	log.Printf("Server is listening on port %s", *port)
	for {
		conn, err := server.Accept()
		if err == nil {
			go checker.HandleClient(conn)
		} else {
			log.Printf("Failed to accept connection: %v", err)
		}
	}
}
