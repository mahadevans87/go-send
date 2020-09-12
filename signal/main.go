package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// PeerInfo Data Model
type PeerInfo struct {
	Token string `json:"token"`
	Sdp   string `json:"sdp"`
	id    string `json:"id"`
	offerer bool
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
			peers = make([]*PeerInfo, 2)
		} else if len(peers) == 2 {
			c.JSON(400, gin.H{
				"error": "Cannot add additional peer to token",
			})
		} else {
			peerID := len(peers) + 1
			peerInfo.id = fmt.Sprint(peerID)
			peers = append(peers, &peerInfo)
			c.JSON(200, gin.H{
				"message": "OK",
				"peerId":  peerID,
			})
		}
	})

	// Set the offer by the first peer.
	r.POST("/offer", func(c *gin.Context) {
		var peerInfo PeerInfo
		c.BindJSON(&peerInfo)
		if peers := tokenPeers[peerInfo.Token]; peers != nil {
			foundPeer := false
			for _, peer := range peers {
				if peer.id == peerInfo.id {
					peer.Sdp = peerInfo.Sdp
					peer.offerer = true
					foundPeer = true
					break
				}
			}
			if foundPeer == true {
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
	r.GET("/offer", func(c *gin.Context) {
		token := c.Query("token")
		peerID := c.Query("id")
		var offerSdp string;
		if peers := tokenPeers[token]; peers != nil {
			foundPeer := false
			for _, peer := range peers {
				if peer.id == peerID {
					foundPeer = true
				}
				if peer.offerer == true {
					offerSdp = peer.Sdp
				}
			}
			if foundPeer == true {
				c.JSON(200, gin.H{
					"message": "OK",
					"offer_sdp": 
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
