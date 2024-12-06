package tcp

import (
	"encoding/json"
	"fmt"
	"log"
)

type Heartbeat struct {
	ID               string   `json:"id"`
	SerialNumber     string   `json:"serial_number"`
	Status           string   `json:"status"`
	MessageType      string   `json:"messageType"`
	ConnectionStatus string   `json:"connection_status"`
	Mode             string   `json:"mode"`
	Temperature      float32  `json:"temperature"`
	Pressure         float32  `json:"pressure"`
	O2Output         float32  `json:"o2_output"`
	O2Concentration  float32  `json:"o2_concentration"`
	CO2Input         float32  `json:"co2_input"`
	CO2Concentration float32  `json:"co2_concentration"`
	PowerConsumption float32  `json:"power_consumption"`
	AlertLevel       string   `json:"alert_level"`
	ErrorCodes       []string `json:"error_codes"`
	ErrorMessages    []string `json:"error_messages"`
	LastMaintenance  string   `json:"last_maintenance"`
	LastCommTime     string   `json:"last_comm_time"`
}

func messageHandler(c *Connection) {
	for msg := range c.msgch {
		//fmt.Println(msg.payload)
		var h Heartbeat
		err := json.Unmarshal(msg.payload, &h)
		if err != nil {
			fmt.Println("Error parsing message:", err)
			continue
		}
		switch h.MessageType {
		case "heartbeat":
			printJSON(h)
		//case "statusUpdate":
		//statusUpdateHandler(m)
		//case "error":
		//errorHandler(m)
		default:
			// Default case if message type is not recognized
			fmt.Println("Received unknown message type:", h.MessageType)
		}
	}

}

func printJSON(message Heartbeat) {
	// Marshal the struct into a JSON byte slice with indentation
	jsonData, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}

	log.Println(string(jsonData))
}
