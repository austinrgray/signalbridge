package tcp

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type Connection struct {
	id     string
	conn   net.Conn
	msgch  chan Message
	quitch chan struct{}
}

type Message struct {
	id          string
	from        string
	payload     []byte
	timestamp   time.Time
	messageType string
}

func newConnection(conn net.Conn) *Connection {
	return &Connection{
		id:     uuid.New().String(),
		conn:   conn,
		msgch:  make(chan Message, 10),
		quitch: make(chan struct{}),
	}
}

func (c *Connection) close() {
	close(c.msgch)
	c.conn.Close()
}
