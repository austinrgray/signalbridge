package tcp

import (
	"fmt"
	"log"
	"net"
)

func handleConnection(conn net.Conn) {
	defer conn.Close() // Ensure the connection is closed when the function exits

	// Placeholder for handling the connection: reading the incoming data
	fmt.Println("New connection from", conn.RemoteAddr())

	// You can implement specific logic here, like reading messages, heartbeats, etc.
	// For now, just print data received from the client.
	buffer := make([]byte, 1024) // Buffer to hold incoming data
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("Error reading data:", err)
			return
		}
		// Print the received message
		fmt.Printf("Received: %s\n", string(buffer[:n]))

		// Simulate a heartbeat message to the device
		_, err = conn.Write([]byte("Heartbeat: Device Active"))
		if err != nil {
			log.Println("Error writing heartbeat:", err)
			return
		}
	}
}
