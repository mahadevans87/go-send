package domain

import (
	"encoding/json"
)

// SignalBaseURL - Signalling Server Base URL
const SignalBaseURL = "http://localhost:8080"

// PeerInfo Data Model
type PeerInfo struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

// ConnectionInfo can be shared between packages
type ConnectionInfo struct {
	Message string `json:"message"`
	ID      string `json:"peerID"`
	Peers   []*PeerInfo
	Token   string
	Mode    string
}

// Message Data Model
type Message struct {
	Type  string
	Token string          `json:"token"`
	From  string          `json:"from"`
	To    string          `json:"to"`
	Data  json.RawMessage `json:"data"`
}

// Messages -> Pending SDP / Candidate Messages from other clients
type Messages struct {
	Message string    `json:"message"`
	Data    []Message `json:"data"`
}
