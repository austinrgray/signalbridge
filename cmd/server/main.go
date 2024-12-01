package main

import (
	"fmt"
	"log"
	"os"
	"signalbridge/internal/tcp"

	//"signalbridge/internal/http"
	//"signalbridge/internal/websocket"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	serverHost := os.Getenv("SERVER_HOST")
	serverPortTCP := os.Getenv("SERVER_PORT_TCP")
	//serverPortHTTP := os.Getenv("SERVER_PORT_HTTP")
	//serverPortWebSocket := os.Getenv("SERVER_PORT_WEBSOCKET")

	tcpServerAddress := fmt.Sprintf("%s:%s", serverHost, serverPortTCP)
	//httpServerAddress := fmt.Sprintf("%s:%s", serverHost, serverPortHTTP)
	//webSocketServerAddress := fmt.Sprintf("%s:%s", serverHost, serverPortWebSocket)

	go tcp.Start(tcpServerAddress)
	//go http.Start(httpServerAddress)
	//go websocket.Start(webSocketServerAddress)

	select {}
}
