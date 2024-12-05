package tcp

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Server struct {
	ln          net.Listener
	listenAddr  string
	connections map[string]*Connection
	connch      chan net.Conn
	quitch      chan struct{}
	wg          sync.WaitGroup
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr:  listenAddr,
		connections: make(map[string]*Connection),
		connch:      make(chan net.Conn, 10),
		quitch:      make(chan struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln

	go s.acceptLoop()

	<-s.quitch
	close(s.connch)

	s.wg.Wait()

	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("server failed to accept error:", err)
		}

		c := newConnection(conn)
		s.connections[c.id] = c
		s.connch <- conn

		fmt.Printf("new connection to the server: %s\n", conn.RemoteAddr())

		go s.readLoop(c)
	}
}

func (s *Server) readLoop(c *Connection) {
	defer c.close()
	buf := make([]byte, 1024)
	for {
		select {
		case <-c.quitch:
			fmt.Printf("Connection %s closed\n", c.id)
		default:
			n, err := c.conn.Read(buf)
			if err != nil {
				fmt.Printf("Error reading from connection %s: %v\n", c.id, err)
				return
			}

			c.msgch <- Message{
				from:      c.conn.RemoteAddr().String(),
				payload:   buf[:n],
				timestamp: time.Now(),
			}

			go messageHandler(c)
		}
	}
}

func (s *Server) Stop() {
	close(s.quitch)
	for _, c := range s.connections {
		c.close()
	}
}

/*
func (s *Server) manageConnections() {
	for conn := range s.connch {
		connection := newConnection(conn)
		s.connections[connection.id] = connection
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			connection.readLoop()
			delete(s.connections, connection.id)
		}()

	}
}*/
