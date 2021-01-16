package domain

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
	ID      int    `json:"peerID"`
	Peers   []*PeerInfo
	Token   string
	Mode    string
}
