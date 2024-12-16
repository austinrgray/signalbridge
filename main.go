package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type config struct {
	tcpAddr       string
	websocketAddr string
	httpAddr      string
}

type supervisor struct {
	tcpPID       *actor.PID
	websocketPID *actor.PID
	httpPID      *actor.PID
}

type tcpServer struct {
	listenAddr string
	listener   net.Listener
	sessions   map[*actor.PID]net.Conn
}

type websocketServer struct {
	listenAddr string
	listener   net.Listener
	sessions   map[*actor.PID]net.Conn
}

type httpServer struct {
	listenAddr string
	listener   net.Listener
	sessions   map[*actor.PID]net.Conn
}

type tcpSession struct {
	conn      net.Conn
	handlePID *actor.PID
	wmch      chan []byte
}

type connAdd struct {
	pid  *actor.PID
	conn net.Conn
}

type connRem struct {
	pid *actor.PID
}

type tcpHandler struct{}

/*
	func loadConfig() config {
		return config{
			tcpAddr:       ":6000",
			websocketAddr: ":7000",
			httpAddr:      ":8000",
		}
	}
*/
func main() {
	e, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		panic(err)
	}

	supervisorPID := e.Spawn(newSupervisor(), "supervisor")

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	<-sigch

	wg := &sync.WaitGroup{}
	e.Poison(supervisorPID, wg)
	wg.Wait()
}

func newSupervisor() actor.Producer {
	return func() actor.Receiver {
		return &supervisor{}
	}
}

func (s *supervisor) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		slog.Info("server started, spawning servers...")

		s.tcpPID = c.SpawnChild(newTCPServer(":6000"), "tcp_server")
		//c.SpawnChild(newWebsocketServer(":7000"), "websocket_server")
		//c.SpawnChild(newHTTPServer(":8000"), "http_server")

	case actor.Stopped:
		slog.Info("Server stopping, shutting down servers.")
	}
}

func newTCPServer(listenAddr string) actor.Producer {
	return func() actor.Receiver {
		return &tcpServer{
			listenAddr: listenAddr,
			sessions:   make(map[*actor.PID]net.Conn),
		}
	}
}

func (s *tcpServer) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Initialized:
		s.initListener()
	case actor.Started:
		slog.Info("server started", "addr", s.listenAddr)
		go s.acceptLoop(c)
	case actor.Stopped:
		// on stop all the childs sessions will automatically get the stop
		// message and close all their underlying connection.
	case *connAdd:
		slog.Debug("added new connection to my map", "addr", msg.conn.RemoteAddr(), "pid", msg.pid)
		s.sessions[msg.pid] = msg.conn
	case *connRem:
		slog.Debug("removed connection from my map", "pid", msg.pid)
		delete(s.sessions, msg.pid)
	}
}

func (s *tcpServer) initListener() {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		slog.Error("failed to load certificates", "err", err)
		panic(err)
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := tls.Listen("tcp", s.listenAddr, config)
	if err != nil {
		slog.Error("failed to start TLS listener", "err", err)
		panic(err)
	}
	s.listener = listener
}

func (s *tcpServer) acceptLoop(c *actor.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			break
		}

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			slog.Error("failed to assert connection as TLS connection")
			conn.Close()
			continue
		}

		if err := tlsConn.Handshake(); err != nil {
			slog.Error("TLS handshake failed", "err", err)
			conn.Close()
			continue
		}

		// Optionally, verify the client certificate if required
		if tlsConn.ConnectionState().PeerCertificates != nil {
			// Inspect peer certificates if you need to enforce mutual TLS, etc.
			// For now, we're not enforcing that.
		}

		pid := c.SpawnChild(newTCPSession(conn), "session", actor.WithID(conn.RemoteAddr().String()))
		c.Send(c.PID(), &connAdd{
			pid:  pid,
			conn: conn,
		})
	}
}

func newTCPSession(conn net.Conn) actor.Producer {
	return func() actor.Receiver {
		return &tcpSession{
			conn: conn,
			wmch: make(chan []byte, 10),
		}
	}
}

func (s *tcpSession) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		slog.Info("new connection", "addr", s.conn.RemoteAddr())
		go s.readLoop(c)
		go s.writeLoop(c)
		s.handlePID = c.SpawnChild(newTCPHandler, "handler")
	case actor.Started:
		slog.Info("connection open", "addr", s.conn.RemoteAddr())
	case actor.Stopped:
		close(s.wmch)
		s.conn.Close()
	case []byte:
		slog.Info("outgoing message", "addr", s.conn.RemoteAddr())
		select {
		case s.wmch <- c.Message().([]byte):
		default:
			slog.Warn("message dropped, channel full")
		}
	}
}

func (s *tcpSession) writeLoop(c *actor.Context) {
	for msg := range s.wmch {
		_, err := s.conn.(*tls.Conn).Write(msg)
		if err != nil {
			slog.Error("conn write error", "err", err)
			break
		}
	}

	c.Send(c.Parent(), &connRem{pid: c.PID()})
}

func (s *tcpSession) readLoop(c *actor.Context) {
	buf := make([]byte, 1024)
	for {
		n, err := s.conn.(*tls.Conn).Read(buf)
		if err != nil {
			slog.Error("conn read error", "err", err)
			break
		}

		// copy shared buffer, to prevent race conditions.
		msg := make([]byte, n)
		copy(msg, buf[:n])
		c.Send(s.handlePID, msg)
	}
	// Loop is done due to error or we need to close due to server shutdown.
	c.Send(c.Parent(), &connRem{pid: c.PID()})
}

func newTCPHandler() actor.Receiver {
	return &tcpHandler{}
}

func (tcpHandler) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case []byte:
		slog.Info("handler received", "message", string(msg))
		c.Send(c.Parent(), msg)
	case actor.Stopped:
		for i := 0; i < 3; i++ {
			fmt.Printf("\r handler stopping in %d", 3-i)
			time.Sleep(time.Second)
		}
		fmt.Println("\n handler stopped")
	default:
		//fmt.Printf("Received message of type: %T\n", msg)
	}
}
