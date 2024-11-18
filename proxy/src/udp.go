// proxy/src/udp.go
package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type UDPServer struct {
	port int
}

func NewUDPServer() *UDPServer {
	return &UDPServer{
		port: 5353,
	}
}

func (s *UDPServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server: %v", err)
	}
	defer conn.Close()

	log.Printf("Started UDP latency simulator on :%d", s.port)

	buffer := make([]byte, 65535)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP: %v", err)
			continue
		}

		go s.handlePacket(conn, buffer[:n], remoteAddr)
	}
}

func (s *UDPServer) handlePacket(conn *net.UDPConn, data []byte, addr *net.UDPAddr) {
	// Get celestial body from hostname
	parts := strings.Split(addr.String(), ".")
	body, bodyName := getCelestialBody(parts[0])
	if body == nil {
		log.Printf("Unknown celestial body in UDP request")
		return
	}

	// Forward destination is Cloudflare DNS
	destAddr, err := net.ResolveUDPAddr("udp", "1.1.1.1:53")
	if err != nil {
		log.Printf("Failed to resolve DNS server: %v", err)
		return
	}

	log.Printf("UDP request via %s (adding %.2f seconds latency)",
		bodyName, calculateLatency(body.Distance*1e6).Seconds())

	// Apply space latency
	latency := calculateLatency(body.Distance * 1e6)
	time.Sleep(latency)

	// Forward to Cloudflare
	forwardConn, err := net.DialUDP("udp", nil, destAddr)
	if err != nil {
		log.Printf("Error connecting to DNS: %v", err)
		return
	}
	defer forwardConn.Close()

	_, err = forwardConn.Write(data)
	if err != nil {
		log.Printf("Error forwarding UDP: %v", err)
		return
	}

	// Get response
	response := make([]byte, 65535)
	forwardConn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, err := forwardConn.Read(response)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return
	}

	// Apply return latency
	time.Sleep(latency)

	_, err = conn.WriteToUDP(response[:n], addr)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}
