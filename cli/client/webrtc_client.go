package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/mahadevans87/go-send/cli/domain"
	"github.com/mahadevans87/go-send/cli/network"

	"github.com/pion/webrtc/v3"
)

// AppError holds generic errors that the app reports.
type AppError struct {
	Cause string
}

func (appError *AppError) Error() string {
	return fmt.Sprintf(appError.Cause)
}

// Connect -> Pass in domain.connectionInfo.
func (pionClient *PionClient) Connect() {
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)
	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// start polling for client messages
	stopPolling := make(chan bool, 1)
	go pollMessages(stopPolling, peerConnection, &candidatesMux, pendingCandidates, pionClient.ConnectionInfo)

	// When an ICE candidate is available send to the other Pion instance
	// the other Pion instance will add this candidate by calling AddICECandidate
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else if onICECandidateErr := signalCandidate(c, pionClient.ConnectionInfo); err != nil {
			panic(onICECandidateErr)
		}
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Create a datachannel with label 'data'
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}
	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())
		// Stop polling for any new messages. Connection has been established
		stopPolling <- true
		close(stopPolling)

		fileBlock := make([]byte, 65535)
		file, err := os.Open("/home/mahadevan/apache-maven.tar.gz")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		for {
			n, err := file.Read(fileBlock)
			if err != nil {
				if err == io.EOF {
					break
				} else {
					panic(err)
				}
			}
			fmt.Printf("\n Reading %v bytes from file ...", n)
			dataErr := dataChannel.Send(fileBlock[:n])
			if dataErr != nil {
				panic(dataErr)
			}

		}
	})

	// Register text message handling
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
	})

	// Create an offer to send to the other process
	if pionClient.ConnectionInfo.Mode == "S" {
		sdpMessage := pionClient.OnReadyToSendOffer(peerConnection)

		// Send our offer to the HTTP server listening in the other process
		payload, err := json.Marshal(sdpMessage)
		if err != nil {
			panic(err)
		}

		if err := sendSDPToPeer(payload); err != nil {
			panic(err)
		}
	}

}

// A handler that processes a SessionDescription given to us from the other Pion process
func handleSDP(peerConnection *webrtc.PeerConnection,
	candidatesMux *sync.Mutex,
	pendingCandidates []*webrtc.ICECandidate,
	incomingMessage domain.Message,
	connectionInfo *domain.ConnectionInfo) error {

	sdp := webrtc.SessionDescription{}
	if sdpString, ok := incomingMessage.Data.(string); ok {
		if err := json.Unmarshal([]byte(sdpString), &sdp); err != nil {
			return err
		}
		if sdpErr := peerConnection.SetRemoteDescription(sdp); sdpErr != nil {
			panic(sdpErr)
		}

		// If this is a peer that is going to send an answer, then

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		for _, c := range pendingCandidates {
			if onICECandidateErr := signalCandidate(c, connectionInfo); onICECandidateErr != nil {
				panic(onICECandidateErr)
			}
		}

	} else {
		return &AppError{"There was an error parsing SDP of peer"}
	}

	return nil
}

// A handler that allows the other Pion instance to send us ICE candidates
// This allows us to add ICE candidates faster, we don't have to wait for STUN or TURN
// candidates which may be slower

func handleICECandidate(peerConnection *webrtc.PeerConnection, candidatesMux *sync.Mutex, pendingCandidates []*webrtc.ICECandidate, incomingMessage domain.Message) error {
	if candidate, ok := incomingMessage.Data.(string); ok {
		if candidateErr := peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(candidate)}); candidateErr != nil {
			return candidateErr
		}
	} else {
		return &AppError{"There was an error parsing ICE Candidate of peer"}
	}
	return nil
}

func parseMessages(peerConnection *webrtc.PeerConnection, candidatesMux *sync.Mutex, pendingCandidates []*webrtc.ICECandidate, connectionInfo *domain.ConnectionInfo) error {
	pendingMessages, err := network.FetchPendingMessages(connectionInfo)
	if err != nil {
		return err
	}
	for _, pendingMessage := range pendingMessages.Data {
		switch pendingMessage.Type {
		case "SDP":
			if err := handleSDP(peerConnection, candidatesMux, pendingCandidates, pendingMessage, connectionInfo); err != nil {
				return err
			}
		case "ICE":
			if err := handleICECandidate(peerConnection, candidatesMux, pendingCandidates, pendingMessage); err != nil {
				return err
			}
		case "OFFER":
		case "ANSWER":
		default:
			return &AppError{"There was an error parsing an incoming message of unkown type"}
		}
	}
	return nil
}

func pollMessages(stopPolling chan bool, peerConnection *webrtc.PeerConnection,
	candidatesMux *sync.Mutex,
	pendingCandidates []*webrtc.ICECandidate,
	connectionInfo *domain.ConnectionInfo) error {
	for {
		select {
		case <-stopPolling:
			return nil
		default:
			if parseErr := parseMessages(peerConnection, candidatesMux, pendingCandidates, connectionInfo); parseErr != nil {
				return parseErr
			} else {
				time.Sleep(2 * time.Second)
				return pollMessages(stopPolling, peerConnection, candidatesMux, pendingCandidates, connectionInfo)
			}

		}
	}
}

func sendSDPToPeer(payload []byte) error {
	resp, err := http.Post(fmt.Sprintf("%s/message", domain.SignalBaseURL), "application/json; charset=utf-8", bytes.NewReader(payload))
	if err != nil {
		return err
	} else if err := resp.Body.Close(); err != nil {
		return err
	}
	return nil
}

func signalCandidate(c *webrtc.ICECandidate, connectionInfo *domain.ConnectionInfo) error {
	//TODO: Send a proper message
	// Wrap it onto our Message object
	iceMessage := domain.Message{
		Data:  c.ToJSON().Candidate,
		From:  (string)(connectionInfo.ID),
		To:    connectionInfo.Peers[0].ID,
		Token: connectionInfo.Token,
		Type:  "ICE",
	}
	payload, err := json.Marshal(iceMessage)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("%s/message", domain.SignalBaseURL),
		"application/json; charset=utf-8", bytes.NewReader(payload))

	if err != nil {
		return err
	}

	if closeErr := resp.Body.Close(); closeErr != nil {
		return closeErr
	}

	return nil
}
