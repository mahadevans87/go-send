package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// PeerInfo Data Model
type PeerInfo struct {
	Token    string       `json:"token"`
	ID       string       `json:"id"`
	Messages chan Message `json:"-"`
}

// Message Data Model
type Message struct {
	Type  string
	Token string `json:"token"`
	From  string `json:"from"`
	To    string `json:"to"`
	Data  string `json:"data"`
}

func main() {
	r := gin.Default()

	// Make a map of token-peerInfos
	tokenPeers := make(map[string][]*PeerInfo)

	r.POST("/register", func(c *gin.Context) {
		var peerInfo PeerInfo
		token := c.Query("token")
		peers := tokenPeers[token]
		if peers == nil {
			peers = make([]*PeerInfo, 0)
			tokenPeers[token] = peers
		}
		if len(peers) == 2 {
			c.JSON(400, gin.H{
				"error": "Cannot add additional peer to token",
			})
		} else {
			peerID := len(peers) + 1
			peerInfo.ID = fmt.Sprint(peerID)
			peerInfo.Token = token
			peerInfo.Messages = make(chan Message, 10)
			peers = append(peers, &peerInfo)
			tokenPeers[token] = peers
			c.JSON(200, gin.H{
				"message": "OK",
				"peerId":  peerInfo.ID,
			})
		}
	})

	r.GET("/peers", func(c *gin.Context) {
		token := c.Query("token")
		peerID := c.Query("id")
		foundPeer := false
		resultPeers := make([]*PeerInfo, 0)
		if peers := tokenPeers[token]; peers != nil {
			for _, peer := range peers {
				if peerID == peer.ID {
					foundPeer = true
				} else {
					resultPeers = append(resultPeers, peer)
				}
			}
			if foundPeer {
				c.JSON(200, gin.H{
					"message": "OK",
					"peers":   resultPeers,
				})
			} else {
				c.JSON(401, gin.H{
					"message": "UnAuthorized",
				})
			}
		}
	})

	// Set the offer by the first peer.
	r.POST("/message", func(c *gin.Context) {
		var message Message
		c.BindJSON(&message)
		if peers := tokenPeers[message.Token]; peers != nil {
			foundSender := false
			foundReceiver := false
			var receiverPeer *PeerInfo
			for _, peer := range peers {
				if peer.ID == message.From {
					foundSender = true
				}
				if peer.ID == message.To {
					foundReceiver = true
					receiverPeer = peer
				}
			}
			if foundSender && foundReceiver {
				receiverPeer.Messages <- message
				c.JSON(200, gin.H{
					"message": "OK. Offer Submitted",
				})

			} else {
				c.JSON(401, gin.H{
					"message": "UnAuthorized. No such peer",
				})
			}
		} else {
			c.JSON(400, gin.H{
				"message": "Invalid Token",
			})
		}
	})

	// Get the offer from the first peer by the other peer(s)
	r.GET("/messages", func(c *gin.Context) {
		token := c.Query("token")
		peerID := c.Query("id")

		if peers := tokenPeers[token]; peers != nil {
			var foundPeer *PeerInfo
			messages := make([]Message, 0)

			for _, peer := range peers {
				if peer.ID == peerID {
					foundPeer = peer
					break
				}
			}
			if foundPeer != nil {
				hasMoreMessages := true
				for {
					if !hasMoreMessages {
						break
					}
					select {
					case msg := <-foundPeer.Messages:
						messages = append(messages, msg)
					default:
						hasMoreMessages = false
					}
				}
				c.JSON(200, gin.H{
					"message": "OK",
					"data":    messages,
				})
			} else {
				c.JSON(401, gin.H{
					"message": "UnAuthorized. No such peer",
				})
			}
		}

	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
