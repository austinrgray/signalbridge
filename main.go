package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
)

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
	initServerLog()

	e, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		panic(err)
	}

	TCPID := e.Spawn(newServerTCP(":9090"), "tcp_server")
	WSPID := e.Spawn(newServerWS(":8080"), "websocket_server")

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	<-sigch

	wg := &sync.WaitGroup{}
	e.Poison(TCPID, wg)
	e.Poison(WSPID, wg)
	wg.Wait()
}

func initServerLog() {
	serverLog, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open server.log: %v", err)
	}
	log.SetOutput(serverLog)                                       // Redirect the default logger
	slog.SetDefault(slog.New(slog.NewTextHandler(serverLog, nil))) // Redirect slog for server
}

type connAdd struct {
	pid  *actor.PID
	conn net.Conn
}

type connRem struct {
	pid *actor.PID
}

type serverTCP struct {
	listenAddress string
	listener      net.Listener
	sessions      map[*actor.PID]net.Conn
	handlerPID    *actor.PID
}

func newServerTCP(listenAddr string) actor.Producer {
	return func() actor.Receiver {
		return &serverTCP{
			listenAddress: listenAddr,
			sessions:      make(map[*actor.PID]net.Conn),
		}
	}
}

func (s *serverTCP) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Initialized:
		slog.Info("initializing listener", "addr", s.listenAddress)
		listener, err := net.Listen("tcp", s.listenAddress)
		if err != nil {
			slog.Error("failed to start TLS listener", "err", err)
			panic(err)
		}
		s.listener = listener
		s.handlerPID = c.SpawnChild(newHandlerTCP, "session_handler")
	case actor.Started:
		slog.Info("server started", "addr", s.listenAddress)
		go s.accepter(c)
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

func (s *serverTCP) accepter(c *actor.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			break
		}
		pid := c.SpawnChild(newSession(conn, s.handlerPID), "session", actor.WithID(conn.RemoteAddr().String()))
		c.Send(c.PID(), &connAdd{
			pid:  pid,
			conn: conn,
		})
	}
}

type session struct {
	conn       net.Conn
	handlerPID *actor.PID
}

func newSession(conn net.Conn, pid *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &session{
			conn:       conn,
			handlerPID: pid,
		}
	}
}

func (s *session) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case *writeTCP:
		slog.Info("writing to", "addr", s.conn.RemoteAddr())
		p, err := msg.packet.MarshalPacket()
		if err != nil {
			slog.Error("packet marshal error", "err", err)
			break
		}
		_, err = s.conn.Write(p)
		if err != nil {
			slog.Error("conn write error", "err", err)
			break
		}
		slog.Info("successfully wrote to", "addr", s.conn.RemoteAddr())
	case actor.Started:
		slog.Info("new connection", "addr", s.conn.RemoteAddr())
		go s.reader(c)
	case actor.Stopped:
		s.conn.Close()
	}
}

func (s *session) reader(c *actor.Context) {
	buf := make([]byte, 1024)
	for {
		n, err := s.conn.Read(buf)
		if err != nil {
			slog.Error("conn read error", "err", err)
			break
		}

		// copy shared buffer, to prevent race conditions.
		msg := make([]byte, n)
		copy(msg, buf[:n])
		c.Send(s.handlerPID, msg)
	}
	// Loop is done due to error or we need to close due to server shutdown.
	c.Send(c.Parent(), &connRem{pid: c.PID()})
}

type writeTCP struct {
	packet packet
}

type handlerTCP struct{}

func newHandlerTCP() actor.Receiver {
	return &handlerTCP{}
}

func (handlerTCP) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case []byte:
		p := packet{
			version: 1,
			data:    msg,
		}
		p.UnmarshalPacket(msg)
		slog.Info("got message to handle:", "msg", string(p.data))
		c.Send(c.Sender(), &writeTCP{packet: p})
	case actor.Stopped:
		for i := 0; i < 3; i++ {
			fmt.Printf("\r handler stopping in %d", 3-i)
			time.Sleep(time.Second)
		}
		fmt.Println("\nhandler stopped")
	}
}

type packet struct {
	version     byte
	instruction byte
	length      uint16
	data        []byte
}

func (p *packet) MarshalPacket() ([]byte, error) {
	length := uint16(len(p.data))
	lengthData := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthData, length)
	b := make([]byte, 0, 1+1+2+length)
	b = append(b, p.version)
	b = append(b, p.instruction)
	b = append(b, lengthData...)
	return append(b, p.data...), nil
}

func (p *packet) UnmarshalPacket(bytes []byte) error {
	HEADER_SIZE := 4
	/*
		if bytes[0] != VERSION {
			return fmt.Errorf("version mismatch %d != %d", bytes[0], VERSION)
		}
	*/
	length := int(binary.BigEndian.Uint16(bytes[2:]))
	end := HEADER_SIZE + length
	if len(bytes) < end {
		return fmt.Errorf("not enough data to parse packet: got %d expected %d", len(bytes), HEADER_SIZE+length)
	}

	p.instruction = bytes[1]
	p.data = bytes[HEADER_SIZE:end]
	return nil
}

type serverWS struct {
	listenAddress string
	context       *actor.Context
	sessions      map[string]*actor.PID
}

type message struct {
	Content string `json:"content"`
	Owner   string `json:"owner"`
}

func newServerWS(lnAddr string) actor.Producer {
	return func() actor.Receiver {
		return &serverWS{
			listenAddress: lnAddr,
			sessions:      make(map[string]*actor.PID),
		}
	}
}

func (s *serverWS) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		log.Println("Server started on port 8080")
		s.serve()
		s.context = ctx
		_ = msg
	case message:
		s.broadcast(ctx.Sender(), msg)
	default:
		fmt.Printf("Server received %v\n", msg)
	}
}

func (s *serverWS) serve() {
	go func() {
		http.HandleFunc("/ws", s.handlerWS)
		http.ListenAndServe(s.listenAddress, nil)
	}()
}

func (s *serverWS) handlerWS(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New connection")
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	username := r.URL.Query().Get("username")
	pid := s.context.SpawnChild(newUser(username, conn, s.context.PID()), username)

	s.sessions[pid.GetID()] = pid
}

func (s *serverWS) broadcast(sender *actor.PID, msg message) {
	for _, pid := range s.sessions {
		if !pid.Equals(sender) {
			s.context.Send(pid, msg)
		}
	}
}

type user struct {
	conn      *websocket.Conn
	context   *actor.Context
	serverPid *actor.PID
	Name      string
}

func newUser(name string, conn *websocket.Conn, serverPid *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &user{
			Name:      name,
			conn:      conn,
			serverPid: serverPid,
		}
	}
}

func (u *user) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		u.context = ctx
		go u.listen()
	case message:
		u.send(&msg)
	case actor.Stopped:
		_ = msg
		u.conn.Close()
	default:
		fmt.Printf("%s received %v\n", u.Name, msg)
	}
}

func (u *user) listen() {
	var msg message
	for {
		if err := u.conn.ReadJSON(&msg); err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		msg.Owner = u.Name

		go u.handleMessage(msg)
	}
}

func (u *user) handleMessage(msg message) {
	switch msg.Content {
	case "exit": // Send exit message to stop the actor and close the websocket connection
		u.context.Engine().Poison(u.context.PID())
	default:
		// Note that this is the server pid, so it will broadcast the message
		u.context.Send(u.serverPid, msg)
	}
}

func (u *user) send(msg *message) {
	if err := u.conn.WriteJSON(msg); err != nil {
		fmt.Printf("Error writing message: %v\n", err)
		return
	}
}
