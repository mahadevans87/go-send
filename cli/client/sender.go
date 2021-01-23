package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mahadevans87/go-send/cli/domain"
	"github.com/pion/webrtc/v3"
)

var pionAdapter *PionAdapter

// PionClient - Implementation of PionAdapter Interface to interact with Pion WebRTC Library
type PionClient struct {
	SenderSourcePath string
	ReceiverDir      string
	ConnectionInfo   *domain.ConnectionInfo
	PeerConnection   *webrtc.PeerConnection
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

// OnDataChannelOpened - Callback when Datachannel is opened - Ref : webrtc_client.go
func (pionClient *PionClient) OnDataChannelOpened(dataChannel *webrtc.DataChannel) {
	fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())

	// Only Send file if mode is "S"
	if pionClient.ConnectionInfo.Mode == "S" {
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
		fmt.Println("\nDone!")

	}
}

// OnDataChannelMessage - Typically used by the receiver mode "R"
func (pionClient *PionClient) OnDataChannelMessage(msg webrtc.DataChannelMessage) {
	file, err := os.OpenFile("/home/mahadevan/apache-maven_copy.tar.gz", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	n, writeErr := file.Write(msg.Data)
	if writeErr != nil {
		panic(writeErr)
	}
	fSyncErr := file.Sync()
	if fSyncErr != nil {
		panic(fSyncErr)
	}
	fmt.Printf("Wrote %v bytes to file\n", n)

}
