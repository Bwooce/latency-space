// proxy/src/websocket.go
package main

import (
	"log"
	"net/http"
	//    "net/url"
	"github.com/gorilla/websocket"
	"strings"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request, body *CelestialBody, destination string) {
	// Parse target URL
	targetURL := destination
	if !strings.HasPrefix(targetURL, "ws") {
		if strings.HasPrefix(targetURL, "https") {
			targetURL = "wss" + targetURL[5:]
		} else {
			targetURL = "ws" + targetURL[4:]
		}
	}

	// Upgrade connection
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer clientConn.Close()

	// Connect to target
	targetConn, _, err := websocket.DefaultDialer.Dial(targetURL, nil)
	if err != nil {
		log.Printf("WebSocket target dial error: %v", err)
		return
	}
	defer targetConn.Close()

	// Handle bidirectional communication
	done := make(chan bool)
	go s.proxyWebSocket(clientConn, targetConn, body, done)
	go s.proxyWebSocket(targetConn, clientConn, body, done)

	<-done
}

func (s *Server) proxyWebSocket(source, target *websocket.Conn, body *CelestialBody, done chan bool) {
	defer func() { done <- true }()

	for {
		messageType, message, err := source.ReadMessage()
		if err != nil {
			return
		}

		// Apply space latency
		latency := calculateLatency(body.Distance * 1e6)
		time.Sleep(latency)

		err = target.WriteMessage(messageType, message)
		if err != nil {
			return
		}
	}
}
