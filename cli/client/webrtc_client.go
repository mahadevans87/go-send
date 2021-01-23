package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

	pionClient.PeerConnection = peerConnection

	// start polling for client messages
	stopPolling := make(chan bool, 1)
	go pionClient.pollMessages(stopPolling, peerConnection, &candidatesMux, pendingCandidates, pionClient.ConnectionInfo)

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

	// Create an offer to send to the other process
	if pionClient.ConnectionInfo.Mode == "S" {
		pionClient.setupDataChannelForSender(stopPolling)
		sdpMessage := pionClient.OnReadyToSendOffer(peerConnection)
		// Send our offer to the HTTP server listening in the other process
		payload, err := json.Marshal(sdpMessage)
		if err != nil {
			panic(err)
		}

		if err := sendSDPToPeer(payload); err != nil {
			panic(err)
		}

	} else if pionClient.ConnectionInfo.Mode == "R" {
		pionClient.setupDataChannelForReceiver(stopPolling)
	}

}

func (pionClient *PionClient) setupDataChannelForSender(stopPolling chan bool) {
	// Create a datachannel with label 'data'
	dataChannel, err := pionClient.PeerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}
	// Register channel opening handling Only for sender
	dataChannel.OnOpen(func() {
		// Stop polling for any new messages. Connection has been established
		stopPolling <- true
		close(stopPolling)
		pionClient.OnDataChannelOpened(dataChannel)
	})

	// Register text message handling
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
	})
}

func (pionClient *PionClient) setupDataChannelForReceiver(stopPolling chan bool) {
	// Register data channel creation handling
	pionClient.PeerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("\nNew DataChannel to receive ...%s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			stopPolling <- true
			close(stopPolling)
			fmt.Printf("\nData channel '%s'-'%d' open. \n", d.Label(), d.ID())
		})

		// Register text message handling
		d.OnMessage(pionClient.OnDataChannelMessage)
	})
}

// A handler that processes a SessionDescription given to us from the other Pion process
func handleSDP(peerConnection *webrtc.PeerConnection,
	candidatesMux *sync.Mutex,
	pendingCandidates []*webrtc.ICECandidate,
	incomingMessage domain.Message,
	pionClient *PionClient) error {

	sdp := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(incomingMessage.Data), &sdp); err != nil {
		return err
	}
	if sdpErr := peerConnection.SetRemoteDescription(sdp); sdpErr != nil {
		panic(sdpErr)
	}

	// Send our answer to the HTTP server listening in the other process
	if pionClient.ConnectionInfo.Mode == "R" {
		sdpMessage := pionClient.OnReadyToSendAnswer(peerConnection)
		payload, err := json.Marshal(sdpMessage)
		if err != nil {
			panic(err)
		}

		if err := sendSDPToPeer(payload); err != nil {
			panic(err)
		}
	}

	candidatesMux.Lock()
	defer candidatesMux.Unlock()

	for _, c := range pendingCandidates {
		if onICECandidateErr := signalCandidate(c, pionClient.ConnectionInfo); onICECandidateErr != nil {
			panic(onICECandidateErr)
		}
	}

	return nil
}

// A handler that allows the other Pion instance to send us ICE candidates
// This allows us to add ICE candidates faster, we don't have to wait for STUN or TURN
// candidates which may be slower

func handleICECandidate(peerConnection *webrtc.PeerConnection, candidatesMux *sync.Mutex, pendingCandidates []*webrtc.ICECandidate, incomingMessage domain.Message) error {
	var candidate string
	if err := json.Unmarshal([]byte(incomingMessage.Data), &candidate); err != nil {
		return &AppError{"There was an error parsing ICE Candidate of peer"}
	} else {
		if candidateErr := peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidate}); candidateErr != nil {
			return candidateErr
		}
	}
	return nil
}

func (pionClient *PionClient) parseMessages(peerConnection *webrtc.PeerConnection, candidatesMux *sync.Mutex, pendingCandidates []*webrtc.ICECandidate, connectionInfo *domain.ConnectionInfo) error {
	pendingMessages, err := network.FetchPendingMessages(connectionInfo)
	if err != nil {
		return err
	}
	for _, pendingMessage := range pendingMessages.Data {
		switch pendingMessage.Type {
		case "SDP":
			if err := handleSDP(peerConnection, candidatesMux, pendingCandidates, pendingMessage, pionClient); err != nil {
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

func (pionClient *PionClient) pollMessages(stopPolling chan bool, peerConnection *webrtc.PeerConnection,
	candidatesMux *sync.Mutex,
	pendingCandidates []*webrtc.ICECandidate,
	connectionInfo *domain.ConnectionInfo) error {
	for {
		select {
		case <-stopPolling:
			return nil
		default:
			if parseErr := pionClient.parseMessages(peerConnection, candidatesMux, pendingCandidates, connectionInfo); parseErr != nil {
				return parseErr
			} else {
				time.Sleep(2 * time.Second)
				return pionClient.pollMessages(stopPolling, peerConnection, candidatesMux, pendingCandidates, connectionInfo)
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
	var candidateBytes []byte
	var err error
	if candidateBytes, err = json.Marshal(c.ToJSON().Candidate); err != nil {
		return err
	}
	iceMessage := domain.Message{
		Data:  candidateBytes,
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
