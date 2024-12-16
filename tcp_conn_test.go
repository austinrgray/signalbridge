package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"testing"
	"time"
)

func startServer() {
	go main()
	time.Sleep(2 * time.Second)
}

func loadTestClient(t *testing.T, clientID int, tlsConfig *tls.Config) {
	// Connect to the server
	conn, err := tls.Dial("tcp", "127.0.0.1:6000", tlsConfig)
	if err != nil {
		t.Errorf("Client %d: Failed to connect to server: %v", clientID, err)
		return
	}
	defer conn.Close()

	// Test message
	testMessage := fmt.Sprintf("Hello from client %d!", clientID)
	_, err = conn.Write([]byte(testMessage))
	if err != nil {
		t.Errorf("Client %d: Failed to send message: %v", clientID, err)
		return
	}

	// Read the response from the server
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Errorf("Client %d: Failed to read response: %v", clientID, err)
		return
	}

	receivedMessage := string(buf[:n])
	if receivedMessage != testMessage {
		t.Errorf("Client %d: Expected response %q, but got %q", clientID, testMessage, receivedMessage)
	} else {
		fmt.Printf("Client %d received: %s\n", clientID, receivedMessage)
	}
}

func LoadTestTCPServer(t *testing.T, numClients int) {
	// Start the server
	startServer()

	// Configure TLS settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Skip verification for testing purposes
	}

	// Simulate multiple clients concurrently
	for i := 0; i < numClients; i++ {
		go loadTestClient(t, i+1, tlsConfig) // Start each client in its own Goroutine
	}

	// Wait a bit for all client connections to complete (can be adjusted for your test scenario)
	time.Sleep(5 * time.Second) // Adjust this to suit the expected response time of your server
}

func TestTCPServerLoad(t *testing.T) {
	LoadTestTCPServer(t, 100)
}
