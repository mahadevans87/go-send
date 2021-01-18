package main

import (
	"github.com/mahadevans87/go-send/cli/client"
	"github.com/mahadevans87/go-send/cli/domain"
	"github.com/mahadevans87/go-send/cli/network"

	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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

func fetchPeerList(peersFound chan bool, connectionInfoPtr *domain.ConnectionInfo) {
	// Fetch PeerInfo
	if peerInfoFetchErr := network.FetchPeerListFromServer(connectionInfoPtr); peerInfoFetchErr != nil {
		log.Fatal(peerInfoFetchErr)
	} else {
		if len(connectionInfoPtr.Peers) == 0 {
			log.Println("Waiting for peers...")
			time.Sleep(2 * time.Second) // Don't flood the server, sleep for a while
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

	if err := network.RegisterToken(*token, &connectionInfo); err != nil {
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
			client.Connect(&connectionInfo)
			//client = Client(&connectionInfo)
			//client.SetSourcePath(*sourcePath)
		case "R":
		default:
			flag.PrintDefaults()
			os.Exit(1)
		}

	}

	select {}

}
