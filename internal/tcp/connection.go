package tcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Connection struct {
	ConnectionID      string
	Status            ConnectionStatus
	RemoteAddress     string
	TCPConnection     net.Conn
	LastCommunication time.Time
	msgch             chan json.RawMessage
	config            Config
	connCtx           context.Context
	cancelCtx         context.CancelFunc
	mu                sync.RWMutex
	wg                sync.WaitGroup
}

type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "CONNECTED"
	StatusDisconnected ConnectionStatus = "DISCONNECTED"
	StatusReconnected  ConnectionStatus = "RECONNECTING"
	StatusInit         ConnectionStatus = "INITIALIZING"
	StatusAuth         ConnectionStatus = "AUTHENTICATING"
	StatusLockout      ConnectionStatus = "LOCKED OUT"
)

func InitializeConnection(conn net.Conn, cfg Config) (*Connection, error) {
	connCtx, cancelCtx := context.WithCancel(context.Background())
	connection := Connection{
		ConnectionID:  uuid.New().String(),
		Status:        StatusInit,
		RemoteAddress: conn.RemoteAddr().String(),
		TCPConnection: conn,
		config:        cfg,
		msgch:         make(chan json.RawMessage, 10),
		connCtx:       connCtx,
		cancelCtx:     cancelCtx,
	}

	err := connection.performHandshake()
	if err != nil {
		connection.Status = StatusDisconnected
		connection.Close()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	connection.Status = StatusConnected
	log.Printf("Connection initialized with ID: %s\n", connection.ConnectionID)

	return &connection, nil
}

func (c *Connection) Start() {
	log.Printf("Connected to: %s", c.RemoteAddress)

	c.wg.Add(1)
	go c.ReadLoop()
	c.wg.Wait()
}

func (c *Connection) ReadLoop() {
	defer c.wg.Done()
	defer c.TCPConnection.Close()

	buf := make([]byte, 2048)
	for {
		select {
		case <-c.connCtx.Done():
			fmt.Println("ReadLoop: Connection context canceled")
			return
		default:
			n, err := c.TCPConnection.Read(buf)
			if err != nil {
				if err == net.ErrClosed || err.Error() == "use of closed network connection" {
					fmt.Println("ReadLoop: Connection closed")
					return
				}
				fmt.Printf("ReadLoop: Read error: %v\n", err)
				return
			}
			if n > 0 {
				log.Println(string(buf[:n]))
				//c.wg.Add(1) go handleMessage(buf[:n])
			}
		}
	}
}

func (c *Connection) Close() {
	if c.cancelCtx != nil {
		c.cancelCtx()
	}

	c.wg.Wait()

	if c.TCPConnection != nil {
		c.TCPConnection.Close()
	}

	close(c.msgch)

	log.Printf("Connection %s has closed\n", c.ConnectionID)
}

func (c *Connection) performHandshake() error {
	ctx, cancel := context.WithTimeout(c.connCtx, 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return fmt.Errorf("handshake timed out")
	default:
		err := c.sendServerPublicKey()
		if err != nil {
			return fmt.Errorf("failed to send public key: %w", err)
		}

		clientToken, err := c.receiveClientToken()
		if err != nil {
			return fmt.Errorf("failed to receive client token: %w", err)
		}

		if !validateClientToken(clientToken) {
			return fmt.Errorf("invalid client token")
		}
	}

	return nil
}

func (c *Connection) sendServerPublicKey() error {
	publicKey := "server-public-key"
	_, err := c.TCPConnection.Write([]byte(publicKey))
	return err
}

func (c *Connection) receiveClientToken() (string, error) {
	buffer := make([]byte, 4096)
	n, err := c.TCPConnection.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}

func validateClientToken(token string) bool {
	return token == "valid-client-token"
}
