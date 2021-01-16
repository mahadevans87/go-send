package main

import (
	"github.com/mahadevans87/go-send/cli/domain"

	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// AppError holds generic errors that the app reports.
type AppError struct {
	Cause string
}

func (appError *AppError) Error() string {
	return fmt.Sprintf(appError.Cause)
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func getJSON(url string, target interface{}) error {
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func registerToken(token string, connectionInfo *domain.ConnectionInfo) error {
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

func fetchPeerListFromServer(connectionInfo *domain.ConnectionInfo) error {
	resp, err := httpClient.Get(fmt.Sprintf("%s/peers?token=%s&id=%d", domain.SignalBaseURL, connectionInfo.Token, connectionInfo.ID))
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

func fetchPeerList(peersFound chan bool, connectionInfoPtr *domain.ConnectionInfo) {
	// Fetch PeerInfo
	if peerInfoFetchErr := fetchPeerListFromServer(connectionInfoPtr); peerInfoFetchErr != nil {
		log.Fatal(peerInfoFetchErr)
	} else {
		if len(connectionInfoPtr.Peers) == 0 {
			log.Println("Waiting for peers...")
			// A better way is to sleep for a while
			fetchPeerList(peersFound, connectionInfoPtr)
		} else {
			peersFound <- true
		}
	}
}

func main() {
	mode := flag.String("mode", "S", "S for send, R for receive. Default is S")
	sourcePath := flag.String("src", "/home/mahadevan/test.txt", "Path of the file to send")
	token := flag.String("token", "", "Token which the sender and receiver must know (Required)")

	flag.Parse()

	if *mode != "S" && *mode != "R" || *token == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	var connectionInfo = domain.ConnectionInfo{}
	if err := registerToken(*token, &connectionInfo); err != nil {
		log.Fatal(err)
	} else {
		// success we have connected
		log.Println(connectionInfo.ID, sourcePath)

		peersAvailable := make(chan bool)
		go fetchPeerList(peersAvailable, &connectionInfo)

		// Wait till peers are available.
		<-peersAvailable
		log.Println(connectionInfo.Peers)

		switch *mode {
		case "S":
			//client = Client(&connectionInfo)
			//client.SetSourcePath(*sourcePath)
		case "R":
		default:
			flag.PrintDefaults()
			os.Exit(1)
		}

	}

}
