package tcp

import (
	"encoding/json"
	"time"
)

type MessageHead struct {
	ConnectionID  string      `json:"connection_id"`
	TransactionID string      `json:"transaction_id"`
	From          string      `json:"from"`
	Type          MessageType `json:"message_type"`
	MessageLength int         `json:"message_length"`
	Timestamp     time.Time   `json:"timestamp"`
}

type Message struct {
	Headers MessageHead    `json:"headers"`
	Payload MessagePayload `json:"payload"`
	Errors  ErrorMessages  `json:"errors,omitempty"`
}

type ErrorMessage struct {
	Code        string    `json:"error_code"`
	Description string    `json:"error_description"`
	AlertLevel  string    `json:"alert_level"`
	Timestamp   time.Time `json:"timestamp"`
}

type MessageType string
type MessagePayload json.RawMessage
type ErrorMessages []ErrorMessage

const (
	MsgTypeHandshake    MessageType = "handshake"
	MsgTypeData         MessageType = "data"
	MsgTypeError        MessageType = "error"
	MsgTypeHeartbeat    MessageType = "heartbeat"
	MsgTypeAck          MessageType = "ack"
	MsgTypeAuthRequest  MessageType = "auth_request"
	MsgTypeAuthResponse MessageType = "auth_response"
	MsgTypeCommand      MessageType = "command"
	MsgTypeResponse     MessageType = "response"
)

func CreateMessage(h MessageHead, p MessagePayload, e ErrorMessages) *Message {
	return &Message{
		Headers: h,
		Payload: p,
		Errors:  e,
	}
}

func CreatMessageHead(connID string, transID string, msgType MessageType) *MessageHead {
	return &MessageHead{
		ConnectionID:  connID,
		TransactionID: transID,
		Type:          msgType,
	}
}

func CreateErrorMessage(errCode string, errDesc string, alertLevel string, t time.Time) *ErrorMessage {
	return &ErrorMessage{
		Code:        errCode,
		Description: errDesc,
		AlertLevel:  alertLevel,
		Timestamp:   t.UTC(),
	}
}
