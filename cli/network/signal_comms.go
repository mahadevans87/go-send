package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mahadevans87/go-send/cli/domain"
)

// AppError holds generic errors that the app reports.
type AppError struct {
	Cause string
}

func (appError *AppError) Error() string {
	return fmt.Sprintf(appError.Cause)
}

// RegisterToken - API that is used to register a client to the signalling server
func RegisterToken(token string, connectionInfo *domain.ConnectionInfo) error {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	resp, err := httpClient.Post(fmt.Sprintf("%s/register?token=%s", domain.SignalBaseURL, token), "", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		decodeErr := json.NewDecoder(resp.Body).Decode(connectionInfo)
		if decodeErr != nil {
			log.Fatal(decodeErr)
			return decodeErr
		}

		// set the token to connectionInfo
		connectionInfo.Token = token
	} else {
		errorMap := make(map[string]string)
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errorMap); decodeErr == nil {
			return &AppError{errorMap["error"]}
		} else {
			return decodeErr
		}

	}
	return err
}

// FetchPeerListFromServer - Fetches PeerList from Server
func FetchPeerListFromServer(connectionInfo *domain.ConnectionInfo) error {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	resp, err := httpClient.Get(fmt.Sprintf("%s/peers?token=%s&id=%s", domain.SignalBaseURL, connectionInfo.Token, connectionInfo.ID))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		type PeerResponse struct {
			Message string             `json:"message"`
			Peers   []*domain.PeerInfo `json:"peers"`
		}
		var peerResponse PeerResponse

		decodeErr := json.NewDecoder(resp.Body).Decode(&peerResponse)
		if decodeErr != nil {
			log.Fatal(decodeErr)
			return decodeErr
		} else {
			// For now there is only one peer. We need to write a proper client later on
			connectionInfo.Peers = peerResponse.Peers
			return nil
		}
	} else {
		return &AppError{"There was an internal server error."}
	}
}

// FetchPendingMessages - Fetches SDP / ICE messages from other clients
func FetchPendingMessages(connectionInfo *domain.ConnectionInfo) (*domain.Messages, error) {
	var httpClient = &http.Client{Timeout: 60 * time.Second}

	resp, err := httpClient.Get(fmt.Sprintf("%s/messages?token=%s&id=%s", domain.SignalBaseURL, connectionInfo.Token, connectionInfo.ID))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var pendingMessages domain.Messages

		decodeErr := json.NewDecoder(resp.Body).Decode(&pendingMessages)
		if decodeErr != nil {
			log.Fatal(decodeErr)
			return nil, decodeErr
		} else {
			// For now there is only one peer. We need to write a proper client later on
			return &pendingMessages, nil
		}
	} else {
		return nil, &AppError{"There was an internal server error."}
	}
}
