package main

import (
	"log"
	"os"
	"os/signal"
	"signalbridge/internal/tcp"
	"syscall"

	//"signalbridge/internal/http"
	//"signalbridge/internal/websocket"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	//host := os.Getenv("SERVER_HOST")
	portTCP := os.Getenv("SERVER_PORT_TCP")
	//portHTTP := os.Getenv("SERVER_PORT_HTTP")
	//portSocket := os.Getenv("SERVER_PORT_WEBSOCKET")
	tcpAddr := ":" + portTCP

	tcpServer := tcp.NewServer(tcpAddr)
	//httpAddr := fmt.Sprintf("%s:%s", host, portHTTP)
	//socketAddr := fmt.Sprintf("%s:%s", host, portSocket)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		if err := tcpServer.Start(); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
		log.Fatal("Server stopped unexpectedly")
	}()

	// Wait for the shutdown signal
	<-shutdown

	// Gracefully stop the server
	tcpServer.Stop()
	log.Println("Server has stopped. Exiting...")
}
