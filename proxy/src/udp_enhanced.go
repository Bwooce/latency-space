// proxy/src/udp_enhanced.go
package main

import (
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type UDPSession struct {
	DestAddr   *net.UDPAddr
	LastActive time.Time
	BytesSent  uint64
	BytesRecv  uint64
	mu         sync.Mutex
}

type UDPProxy struct {
	sessions map[string]*UDPSession
	mu       sync.RWMutex
	timeout  time.Duration
}

func NewUDPProxy() *UDPProxy {
	proxy := &UDPProxy{
		sessions: make(map[string]*UDPSession),
		timeout:  5 * time.Minute,
	}

	// Start session cleanup
	go proxy.cleanupSessions()
	return proxy
}

func (p *UDPProxy) cleanupSessions() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for key, session := range p.sessions {
			if now.Sub(session.LastActive) > p.timeout {
				delete(p.sessions, key)
			}
		}
		p.mu.Unlock()
	}
}

func (p *UDPProxy) handlePacket(conn *net.UDPConn, data []byte, addr *net.UDPAddr, body *CelestialBody) {
	sessionKey := addr.String()

	p.mu.Lock()
	session, exists := p.sessions[sessionKey]
	if !exists {
		// New session: first packet contains destination
		if len(data) < 6 {
			p.mu.Unlock()
			return
		}

		destPort := binary.BigEndian.Uint16(data[0:2])
		ipBytes := data[2:6]
		destIP := net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
		destAddr := &net.UDPAddr{
			IP:   destIP,
			Port: int(destPort),
		}

		session = &UDPSession{
			DestAddr:   destAddr,
			LastActive: time.Now(),
		}
		p.sessions[sessionKey] = session
		data = data[6:] // Remove header
	}
	p.mu.Unlock()

	session.mu.Lock()
	session.LastActive = time.Now()
	session.BytesSent += uint64(len(data))
	session.mu.Unlock()

	// Apply space latency
	latency := calculateLatency(body.Distance * 1e6)
	time.Sleep(latency)

	// Apply bandwidth limiting
	delay := time.Duration(float64(len(data)*8)/float64(body.BandwidthKbps)*1000) * time.Millisecond
	time.Sleep(delay)

	// Forward to destination
	forwardConn, err := net.DialUDP("udp", nil, session.DestAddr)
	if err != nil {
		return
	}
	defer forwardConn.Close()

	_, err = forwardConn.Write(data)
	if err != nil {
		return
	}

	// Handle response
	response := make([]byte, 65535)
	forwardConn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, err := forwardConn.Read(response)
	if err != nil {
		return
	}

	session.mu.Lock()
	session.BytesRecv += uint64(n)
	session.mu.Unlock()

	// Apply return latency
	time.Sleep(latency)

	conn.WriteToUDP(response[:n], addr)
}
