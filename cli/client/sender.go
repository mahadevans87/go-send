package client

import (
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

	// Wrap it onto our Message object
	sdpMessage := domain.Message{
		Data:  offer.SDP,
		From:  pionClient.ConnectionInfo.ID,
		To:    pionClient.ConnectionInfo.Peers[0].ID,
		Token: pionClient.ConnectionInfo.Token,
		Type:  "SDP",
	}
	return sdpMessage
}
