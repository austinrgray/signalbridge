package tcp

import (
	"log"
	"net"
	"time"
)

// Connection wraps a net.Conn and adds extra functionality to handle the device state, heartbeat, and errors.
type Connection struct {
	conn          net.Conn
	deviceAddress string // Can be used to store specific device information like IP, Serial number, etc.
	lastHeartbeat time.Time
}

// NewConnection initializes a new Connection object.
func NewConnection(conn net.Conn, deviceAddress string) *Connection {
	return &Connection{
		conn:          conn,
		deviceAddress: deviceAddress,
		lastHeartbeat: time.Now(),
	}
}

// Close gracefully closes the connection.
func (c *Connection) Close() {
	err := c.conn.Close()
	if err != nil {
		log.Println("Error closing connection for", c.deviceAddress, ":", err)
	} else {
		log.Println("Connection closed for", c.deviceAddress)
	}
}

// SendHeartbeat sends a heartbeat message to the connected device.
func (c *Connection) SendHeartbeat() error {
	heartbeatMessage := []byte("Heartbeat: Device Active")
	_, err := c.conn.Write(heartbeatMessage)
	if err != nil {
		return err
	}

	// Update the last heartbeat timestamp after sending the message
	c.lastHeartbeat = time.Now()
	log.Printf("Heartbeat sent to device %s", c.deviceAddress)
	return nil
}

// IsConnectionActive checks if the connection is still active.
func (c *Connection) IsConnectionActive() bool {
	// This is a basic check; can be enhanced to check if the connection is alive.
	// For example, you could implement a keep-alive message exchange here.

	// If the last heartbeat is too old, consider the connection inactive.
	if time.Since(c.lastHeartbeat) > 60*time.Second {
		log.Printf("Connection to %s timed out", c.deviceAddress)
		return false
	}

	return true
}

// HandleCommunication handles incoming communication with the device.
func (c *Connection) HandleCommunication() {
	// Example of reading from the connection
	buffer := make([]byte, 1024)

	for {
		// Read data from the device
		n, err := c.conn.Read(buffer)
		if err != nil {
			log.Println("Error reading data from", c.deviceAddress, ":", err)
			break
		}

		// Process the received data (e.g., log, parse, handle commands, etc.)
		receivedMessage := string(buffer[:n])
		log.Printf("Received from %s: %s", c.deviceAddress, receivedMessage)

		// Example: Send a response to the device
		_, err = c.conn.Write([]byte("Response: OK"))
		if err != nil {
			log.Println("Error sending response to", c.deviceAddress, ":", err)
			break
		}
	}
}
