// proxy/src/websocket.go
package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader defines the WebSocket upgrader configuration.
// CheckOrigin allows all origins (use with caution, consider specific origins in production).
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for now. In production, you might want to
		// restrict this to specific origins based on the request 'r'.
		return true
	},
}

// NOTE: WebSocket proxying is planned but not yet implemented.
// Future implementation will handle WebSocket connections, applying latency
// similar to HTTP/SOCKS requests. This requires careful handling of the
// persistent connection, message framing, and applying delays without
// blocking other operations excessively.
