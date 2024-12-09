package tcp

import (
	"log"
	"net"
	"sync"
)

type Server struct {
	Config      Config
	ListenAddr  string
	Listener    net.Listener
	Connections map[string]*Connection
	mu          sync.RWMutex
	wg          sync.WaitGroup
}

func NewServer() *Server {
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to load config %s", err)
	}

	addr := string(config.Env.TCPServerHost + config.Env.TCPServerPort)
	return &Server{
		ListenAddr:  addr,
		Connections: make(map[string]*Connection),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.Listener = ln

	log.Printf("Server listening at: %s", s.ListenAddr)

	s.AcceptLoop()
	return nil
}

func (s *Server) Stop() {
	log.Printf("Shutting down the server at: %s", s.ListenAddr)
	s.Listener.Close()

	s.mu.Lock()
	for _, conn := range s.Connections {
		conn.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Server) AcceptLoop() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				log.Println("AcceptLoop: Listener has been closed. Exiting...")
				return
			}
			log.Printf("AcceptLoop: Error accepting connection: %v", err)
			continue
		}

		s.wg.Add(1)
		go func(conn net.Conn) {
			defer s.wg.Done()

			connection, err := InitializeConnection(conn, s.Config)
			if err != nil {
				log.Printf("AcceptLoop: Error initializing connection: %v", err)
				return
			}

			s.mu.Lock()
			s.Connections[connection.ConnectionID] = connection
			s.mu.Unlock()

			connection.Start()

			s.mu.Lock()
			delete(s.Connections, connection.ConnectionID)
			s.mu.Unlock()

			connection.Close()
			log.Printf("Connection %s closed", conn.RemoteAddr())
		}(conn)
	}
}
