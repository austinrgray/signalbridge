package tcp

import (
	"fmt"
	"log"
	"net"
)

func Start(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Error starting TCP server: ", err)
	}
	defer listener.Close() // Ensure the listener is closed when the function exits

	fmt.Println("TCP server started on", address)

	// Accept and handle incoming connections in an infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		// Handle each incoming connection in a separate goroutine
		go handleConnection(conn)
	}
}
