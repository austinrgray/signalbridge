package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type testShip struct {
	id       string   `json:"ship_id"`
	heading  vector   `json:"heading"`
	fuel     uint8    `json:"fuel"`
	hbCount  uint8    `json:"hbCount"`
	hbInt    uint8    `json:"hbInt"`
	manifest manifest `json:"manifest"`
}

type manifest struct {
	id         string `json:"manifest_id"`
	weight     uint8  `json:"weight"`
	souls      uint8  `json:"souls"`
	provisions uint8  `json:"provisions"`
	lithium    uint8  `json:"lithium"`
	iron       uint8  `json:"iron"`
}

func newTestShip() *testShip {
	m := manifest{
		id:         uuid.New().String(),
		weight:     uint8(rand.Intn(256)),
		souls:      uint8(rand.Intn(256)),
		provisions: uint8(rand.Intn(256)),
		lithium:    uint8(rand.Intn(256)),
		iron:       uint8(rand.Intn(256)),
	}
	return &testShip{
		id:       uuid.New().String(),
		heading:  newVector(),
		fuel:     100,
		hbCount:  0,
		hbInt:    3,
		manifest: m,
	}
}

func (s testShip) shipRun(wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()

	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		t.Errorf("Client %s: Failed to connect to server: %v", s.id, err)
		return
	}
	defer conn.Close()

	maxTicks := 100
	tickCount := 0

	ticker := time.NewTicker(time.Duration(s.hbInt) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tickCount++

			if tickCount >= maxTicks {
				fmt.Println("Reached 100 ticks, stopping ticker.")
				return // This will exit the function, stopping the loop
			}

			data := s.createHeartbeatData()
			if err != nil {
				t.Errorf("Client %s: Failed to create hb data: %v", s.id, err)
				return
			}
			hbPacket := createHeartbeatPacket(data)
			bytes, err := hbPacket.MarshalPacket()
			if err != nil {
				t.Errorf("Client %s: Failed to marshal packet data: %v", s.id, err)
				return
			}

			_, err = conn.Write(bytes)
			if err != nil {
				t.Errorf("Client %s: Failed to send message: %v", s.id, err)
				return
			}
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil && err != io.EOF {
				t.Errorf("Client %s: Failed to read response: %v", s.id, err)
				return
			}

			rb := buf[:n]
			if string(rb) != string(bytes) {
				t.Errorf("Client %s: Expected response %q, but got %q", s.id, string(bytes), string(rb))
			} else {
				t.Logf("Client %s: Successfully received: %s", s.id, string(rb))
			}
		}
	}
}

func createHeartbeatPacket(b []byte) *packet {
	return &packet{
		version:     1,
		instruction: 0x01,
		data:        b,
	}
}

func (s *testShip) createHeartbeatData() []byte {
	s.hbCount = s.hbCount + 1
	s.changeHeading()

	hbJSON, err := json.Marshal(s)
	if err != nil {
		log.Fatalf("Error marshalling testShip: %v", err)
	}

	return hbJSON
}

func newVector() vector {
	return vector{
		x: randFloat(-100, 100),
		y: randFloat(-100, 100),
		z: randFloat(-100, 100),
	}.normalize()
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

type vector struct {
	x float64 `json:"x"`
	y float64 `json:"y"`
	z float64 `json:"z"`
}

func (v vector) velocity() float64 {
	return math.Sqrt(v.x*v.x + v.y*v.y + v.z*v.z)
}

func (v vector) normalize() vector {
	velocity := v.velocity()
	if velocity == 0 {
		return vector{0, 0, 0} // Avoid division by zero
	}
	return vector{v.x / velocity, v.y / velocity, v.z / velocity}
}

func (s *testShip) changeHeading() {
	if s.fuel <= 0 {
		fmt.Println("Out of fuel! Cannot change heading.")
		return
	}
	newHeading := newVector()
	fuelCost := newHeading.velocity() * 0.01

	if s.fuel < uint8(fuelCost) { // Convert fuelCost to uint8 before comparison
		fmt.Println("Not enough fuel to change heading.")
		return
	}

	s.heading = newHeading
	s.fuel -= uint8(fuelCost)

	fmt.Printf("Heading changed to: x=%.2f, y=%.2f, z=%.2f\n", s.heading.x, s.heading.y, s.heading.z)
	fmt.Printf("Remaining fuel: %.2f\n", float64(s.fuel))
}

/*
	func startServer(wg *sync.WaitGroup) {
		defer wg.Done()
		main()
	}
*/
func concurrentClientsRun(t *testing.T, numClients int) {
	// Start the server
	//serverWg := &sync.WaitGroup{}
	//serverWg.Add(1)
	//go startServer(serverWg)

	//time.Sleep(1 * time.Second) // Allow server to start

	// Start clients
	clientWg := &sync.WaitGroup{}
	for i := 0; i < numClients; i++ {
		clientWg.Add(1)
		ship := newTestShip()
		go ship.shipRun(clientWg, t)
		time.Sleep(time.Duration(rand.Intn(3-1+1) + 1))
	}

	clientWg.Wait()

	// Send SIGTERM to the server to stop gracefully
	/*
		t.Log("Sending SIGTERM to server...")
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Logf("Failed to find process: %v", err)
		}
		if err := p.Signal(syscall.SIGTERM); err != nil {
			t.Logf("Failed to send SIGTERM signal: %v", err)
		}

		// Wait for server shutdown
		serverWg.Wait()
		t.Log("Server exited gracefully.")
	*/
}

func TestTCP(t *testing.T) {
	concurrentClientsRun(t, 1)
}
