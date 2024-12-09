package main

import (
	"log"
	"os"
	"os/signal"
	"signalbridge/internal/tcp"
	"syscall"
	//"signalbridge/internal/http"
	//"signalbridge/internal/websocket"
)

func main() {
	tcpServer := tcp.NewServer()
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := tcpServer.Start(); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
		log.Fatal("Server stopped unexpectedly.")
	}()

	<-shutdown
	log.Println("Shutdown signal received, stopping server...")

	tcpServer.Stop()
	log.Println("Server has stopped. Exiting...")

}
