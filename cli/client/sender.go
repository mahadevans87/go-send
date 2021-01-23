package client

import (
	"encoding/json"

	"github.com/mahadevans87/go-send/cli/domain"
	"github.com/pion/webrtc/v3"
)

var pionAdapter *PionAdapter

// PionClient - Implementation of PionAdapter Interface to interact with Pion WebRTC Library
type PionClient struct {
	ConnectionInfo *domain.ConnectionInfo
	PeerConnection *webrtc.PeerConnection
}

func (pionClient *PionClient) updatePeerConnection(conn *webrtc.PeerConnection) {
	pionClient.PeerConnection = conn
}

// OnReadyToSendOffer - Interface implementation of PionAdapter
func (pionClient *PionClient) OnReadyToSendOffer(peerConn *webrtc.PeerConnection) domain.Message {

	// Store peerConnection so that we can use it later.
	pionClient.updatePeerConnection(peerConn)

	offer, err := pionClient.PeerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = pionClient.PeerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}
	var offerBytes []byte
	if offerBytes, err = json.Marshal(offer); err != nil {
		panic(err)
	}
	// Wrap it onto our Message object
	sdpMessage := domain.Message{
		Data:  offerBytes,
		From:  pionClient.ConnectionInfo.ID,
		To:    pionClient.ConnectionInfo.Peers[0].ID,
		Token: pionClient.ConnectionInfo.Token,
		Type:  "SDP",
	}
	return sdpMessage
}

// OnReadyToSendAnswer - For the receiver primarily.
func (pionClient *PionClient) OnReadyToSendAnswer(peerConnection *webrtc.PeerConnection) domain.Message {
	pionClient.updatePeerConnection(peerConnection)

	// If this is a peer that is going to send an answer, then
	// Create an answer to send to the other process
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	var answerBytes []byte
	if answerBytes, err = json.Marshal(answer); err != nil {
		panic(err)
	}

	// Wrap it onto our Message object
	sdpMessage := domain.Message{
		Data:  answerBytes,
		From:  pionClient.ConnectionInfo.ID,
		To:    pionClient.ConnectionInfo.Peers[0].ID,
		Token: pionClient.ConnectionInfo.Token,
		Type:  "SDP",
	}
	return sdpMessage
}
