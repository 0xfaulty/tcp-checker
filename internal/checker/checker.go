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
	TCPKeepAlivePeriod = 5 * time.Second        // TCP Keep-Alive period
	MinReadTimeout     = 150 * time.Millisecond // Minimum allowed read timeout (1.5×KeepaliveInterval)
	MaxReadTimeout     = 3 * time.Second        // Maximum allowed read timeout
	InitialRTT         = 100 * time.Millisecond // Initial RTT estimate
	RTTSmoothingFactor = 0.5                    // Smoothing factor for RTT adjustment
)

// KeepaliveMessage is a single newline character
var KeepaliveMessage = []byte("\n")

// LastDisconnectTime stores the last disconnect timestamp for each client (by IP)
var (
	LastDisconnectTime  = make(map[string]time.Time)
	lastDisconnectMutex sync.Mutex // Protects access to LastDisconnectTime
)

// HandleClient processes an incoming connection and periodically sends keepalive messages
func HandleClient(conn net.Conn) {
	// Get IP without port by IP
	clientAddr := conn.RemoteAddr().(*net.TCPAddr)
	clientIP := clientAddr.IP.String()

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
	estimatedRTT := InitialRTT

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
			for {
				// Compute the adaptive timeout as estimatedRTT*2, bounded by MinReadTimeout and MaxReadTimeout
				adaptiveTimeout := estimatedRTT * 2
				if adaptiveTimeout < MinReadTimeout {
					adaptiveTimeout = MinReadTimeout
				}
				if adaptiveTimeout > MaxReadTimeout {
					adaptiveTimeout = MaxReadTimeout
				}

				_ = conn.SetReadDeadline(time.Now().Add(adaptiveTimeout))
				start := time.Now()
				_, err := conn.Read(buf)
				if err != nil {
					log.Printf("Connection with %s lost: %v", serverAddr, err)
					break
				}
				// Measuring RTT by Read execution time
				rtt := time.Since(start)
				estimatedRTT = time.Duration(float64(estimatedRTT)*(1-RTTSmoothingFactor) + float64(rtt)*RTTSmoothingFactor)
			}
		}()
	}
}
