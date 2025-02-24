package checker

import (
	"log"
	"net"
	"sync"
	"time"
)

// Constants for timing configurations
const (
	KeepaliveInterval  = 100 * time.Millisecond // Interval between keepalive messages
	ReadTimeoutFactor  = 1.5                    // Factor for read deadline (1.5× KeepaliveInterval)
	ReadTimeout        = time.Duration(float64(KeepaliveInterval) * ReadTimeoutFactor)
	TCPKeepAlivePeriod = 5 * time.Second // TCP Keep-Alive period
)

// KeepaliveMessage is a single newline character
var KeepaliveMessage = []byte("\n")

// LastDisconnectTime stores the last disconnect timestamp for each client
var (
	LastDisconnectTime  = make(map[string]time.Time) // Now based only on IP, not port
	lastDisconnectMutex sync.Mutex                   // Protects access to LastDisconnectTime
)

// HandleClient processes an incoming connection and periodically sends keepalive messages
func HandleClient(conn net.Conn) {
	clientAddr := conn.RemoteAddr().(*net.TCPAddr) // Get IP without port
	clientIP := clientAddr.IP.String()             // Use only IP

	// Check if the client had a previous disconnection
	lastDisconnectMutex.Lock()
	lastDisconnect, exists := LastDisconnectTime[clientIP]
	lastDisconnectMutex.Unlock()

	if exists && !lastDisconnect.IsZero() {
		downtime := time.Since(lastDisconnect)
		log.Printf("Connection from %s restored, downtime: %v", clientIP, downtime)
	}

	log.Printf("Connection established from %s", clientIP)

	// Remove from map to avoid logging downtime twice
	lastDisconnectMutex.Lock()
	delete(LastDisconnectTime, clientIP)
	lastDisconnectMutex.Unlock()

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}

		// Store the last disconnect time
		lastDisconnectMutex.Lock()
		LastDisconnectTime[clientIP] = time.Now()
		lastDisconnectMutex.Unlock()
		log.Printf("Connection with %s lost", clientIP)
	}()

	// Disable Nagle Algorithm for instant sending
	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		_ = tcpConn.SetNoDelay(true)
	}

	for {
		_, err := conn.Write(KeepaliveMessage) // Send minimal keepalive
		if err != nil {
			return // Exit the function and trigger defer
		}
		time.Sleep(KeepaliveInterval) // Send every KeepaliveInterval (fast detection)
	}
}

// RunClient establishes a persistent connection and expects keepalive messages
func RunClient(serverAddr string) {
	var lastDisconnect time.Time
	connected := false

	for {
		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			if connected {
				log.Printf("Connection to %s lost: %v", serverAddr, err)
				lastDisconnect = time.Now()
				connected = false
			}
			time.Sleep(KeepaliveInterval)
			continue
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if ok {
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(TCPKeepAlivePeriod) // Shorter KeepAlive
			_ = tcpConn.SetNoDelay(true)                       // Disable Nagle Algorithm
		}

		if !connected {
			if !lastDisconnect.IsZero() {
				downtime := time.Since(lastDisconnect)
				log.Printf("Connection to %s restored, downtime: %v", serverAddr, downtime)
			} else {
				log.Printf("Connected to %s for the first time", serverAddr)
			}
			connected = true
		}

		func() {
			defer func() {
				if err := conn.Close(); err != nil {
					log.Printf("Error closing client connection: %v", err)
				}
			}()

			buf := make([]byte, 1)
			lastReceived := time.Now()

			for {
				_ = conn.SetReadDeadline(time.Now().Add(ReadTimeout))
				_, err := conn.Read(buf)

				if err != nil {
					// Check if the last received message was more than KeepaliveInterval ago
					if time.Since(lastReceived) > KeepaliveInterval {
						log.Printf("Connection with %s lost: %v", serverAddr, err)
						break
					}
				} else {
					lastReceived = time.Now() // Update last received timestamp
				}
			}
		}()
	}
}
