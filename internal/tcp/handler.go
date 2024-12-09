package tcp

/*
import (
	"encoding/json"
	"fmt"
	"log"
)

func handleMessage(c *Connection, msg Message[any]) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch msg.MsgType {
	case "HEARTBEAT":
		heartbeat, ok := msg.Payload.(Heartbeat)
		if !ok {
			log.Printf("Error: Payload is not of type Heartbeat")
			return
		}
		processHeartbeat(msg, heartbeat)

	case "INSTRUCTION":
		instruction, ok := msg.Payload.(Instruction)
		if !ok {
			log.Printf("Error: Payload is not of type Instruction")
			return
		}
		processInstruction(instruction)
	}
}

func processHeartbeat(msg Message[any], hb Heartbeat) {
	messageJSON, err := marshalMessage(Message[Heartbeat]{Payload: messageJSON, MsgType: "HEARTBEAT"})
	if err != nil {
		log.Printf("Error marshalling heartbeat: %v", err)
	}
	log.Println(string(*messageJSON))
}

func processInstruction(in Instruction) {
	messageJSON, err := marshalMessage(Message[Instruction]{Payload: in, MsgType: "INSTRUCTION"})
	if err != nil {
		log.Printf("Error marshalling instruction: %v", err)
	}
	log.Println(string(*messageJSON))
}

func unMarshalMessage(messageJSON []byte) (*Message[any], error) {
	var message Message[any]
	err := json.Unmarshal(messageJSON, &message)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling command: %v", err)
	}
	fmt.Printf("MessageType: %s", message.MsgType)

	switch message.MsgType {
	case "HEARTBEAT":
		var payload Heartbeat
		err := json.Unmarshal(messageJSON, &payload)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling heartbeat payload: %v", err)
		}
		fmt.Printf("HeartbeatPayload:DeviceID: %s\n", payload.ID)
		message.Payload = payload
		fmt.Printf("Payload struct: %s\n", message.Payload)
	case "INSTRUCTION":
		var payload Instruction
		err := json.Unmarshal(messageJSON, &payload)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling instruction payload: %v", err)
		}
		message.Payload = payload
	default:
		return nil, fmt.Errorf("Unknown message type: %s", message.MsgType)
	}

	return &message, nil
}

func marshalMessage[T any](message Message[T]) (*[]byte, error) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling payload: %v", err)
	}
	return &messageJSON, nil
}
*/
