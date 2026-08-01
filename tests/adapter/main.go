package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	ProtocolVersion = "1.0"
	// MaxMessageSize sets the upper limit for incoming JSON payload size to 1MB.
	MaxMessageSize  = 1024 * 1024
)

// Message Types
const (
	MsgTypeHandshake    = "handshake"
	MsgTypeRequest      = "request"
	MsgTypeResponse     = "response"
	MsgTypeShutdown     = "shutdown"
	MsgTypeSimulateDelay = "simulate_delay"
	MsgTypeError        = "error"
)

// Message defines the standard JSON structure for both requests and responses.
type Message struct {
	Version string                 `json:"version"`
	ID      string                 `json:"id,omitempty"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

func main() {
	// Disable logging to stdout to keep JSON protocol clean.
	log.SetOutput(os.Stderr)

	scanner := bufio.NewScanner(os.Stdin)

	// Set custom buffer size to enforce the maximum supported message size.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, MaxMessageSize)

	for scanner.Scan() {
		line := scanner.Text()

		var req Message
		var resp Message

		err := json.Unmarshal([]byte(line), &req)
		if err != nil {
			sendErrorResponse("", "invalid json format")
			continue
		}

		// Validate required fields
		if req.Version == "" || req.Type == "" {
			sendErrorResponse(req.ID, "missing required fields: 'version' and 'type' must be provided")
			continue
		}

		if req.Version != ProtocolVersion {
			sendErrorResponse(req.ID, fmt.Sprintf("unsupported protocol version: expected %s, got %s", ProtocolVersion, req.Version))
			continue
		}

		resp = Message{
			Version: ProtocolVersion,
			ID:      req.ID,
		}

		switch req.Type {
		case MsgTypeHandshake:
			resp.Type = MsgTypeHandshake
			resp.Payload = map[string]interface{}{
				"status": "ready",
			}
			sendResponse(resp)

		case MsgTypeShutdown:
			resp.Type = MsgTypeShutdown
			resp.Payload = map[string]interface{}{
				"status": "closing",
			}
			sendResponse(resp)
			// Gracefully return from main rather than invoking os.Exit(0)
			return

		case MsgTypeRequest:
			if req.ID == "" {
				sendErrorResponse("", "missing 'id' for request message")
				continue
			}
			resp.Type = MsgTypeResponse
			resp.Payload = req.Payload
			sendResponse(resp)

		case MsgTypeSimulateDelay:
			if req.ID == "" {
				sendErrorResponse("", "missing 'id' for simulate_delay message")
				continue
			}
			resp.Type = MsgTypeResponse
			if ms, ok := req.Payload["ms"].(float64); ok {
				time.Sleep(time.Duration(ms) * time.Millisecond)
			}
			resp.Payload = req.Payload
			sendResponse(resp)

		default:
			sendErrorResponse(req.ID, fmt.Sprintf("unknown message type: %s", req.Type))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error (possible message size limit exceeded): %v\n", err)
	}
}

func sendErrorResponse(id string, errMsg string) {
	sendResponse(Message{
		Version: ProtocolVersion,
		ID:      id,
		Type:    MsgTypeError,
		Error:   errMsg,
	})
}

func sendResponse(resp Message) {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v\n", err)
		return
	}
	// Print followed by newline to adhere to ND-JSON protocol.
	fmt.Println(string(respBytes))
}
