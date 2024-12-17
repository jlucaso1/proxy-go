package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TCPProxy struct {
	TargetHost string
	TargetPort int
	AllowedIPs []string
	Logger     *log.Logger
}

func loadEnvConfig() *TCPProxy {
	// Target configuration
	targetHost := os.Getenv("TARGET_HOST")
	if targetHost == "" {
		targetHost = "127.0.0.1" // Default value
	}

	targetPortStr := os.Getenv("TARGET_PORT")
	targetPort, err := strconv.Atoi(targetPortStr)
	if err != nil || targetPort == 0 {
		targetPort = 9000 // Default port
	}

	// IP Filtering
	allowedIPsStr := os.Getenv("ALLOWED_IPS")
	var allowedIPs []string
	if allowedIPsStr != "" {
		// Split by comma, trim whitespace
		allowedIPs = strings.Split(allowedIPsStr, ",")
		for i, ip := range allowedIPs {
			allowedIPs[i] = strings.TrimSpace(ip)
		}
	}

	// Proxy configuration
	listenPortStr := os.Getenv("LISTEN_PORT")
	listenPort, err := strconv.Atoi(listenPortStr)
	if err != nil || listenPort == 0 {
		listenPort = 9001 // Default listen port
	}

	proxy := &TCPProxy{
		TargetHost: targetHost,
		TargetPort: targetPort,
		AllowedIPs: allowedIPs,
		Logger:     log.New(os.Stdout, "TCP Proxy: ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	// Log configuration for debugging
	proxy.Logger.Printf("Proxy Configuration:")
	proxy.Logger.Printf("Target: %s:%d", proxy.TargetHost, proxy.TargetPort)
	proxy.Logger.Printf("Allowed IPs: %v", proxy.AllowedIPs)
	proxy.Logger.Printf("Listen Port: %d", listenPort)

	return proxy
}

func (p *TCPProxy) isIPAllowed(clientIP string) bool {
	// If no IPs are specified, allow all
	if len(p.AllowedIPs) == 0 {
		return true
	}

	for _, allowedIP := range p.AllowedIPs {
		if clientIP == allowedIP {
			return true
		}
	}
	return false
}

func (p *TCPProxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Get client IP
	clientIP := clientConn.RemoteAddr().(*net.TCPAddr).IP.String()

	// Check IP filtering
	if !p.isIPAllowed(clientIP) {
		p.Logger.Printf("Blocked connection from unauthorized IP: %s", clientIP)
		return
	}

	p.Logger.Printf("Accepted connection from: %s", clientIP)

	// Connect to target
	targetAddr := fmt.Sprintf("%s:%d", p.TargetHost, p.TargetPort)
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		p.Logger.Printf("Failed to connect to target: %v", err)
		return
	}
	defer targetConn.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(targetConn, clientConn)
		targetConn.(*net.TCPConn).CloseWrite()
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
		clientConn.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
}

func (p *TCPProxy) Start(listenPort int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	defer listener.Close()

	p.Logger.Printf("TCP Proxy listening on port %d", listenPort)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			p.Logger.Printf("Error accepting connection: %v", err)
			continue
		}

		go p.handleConnection(clientConn)
	}
}

func main() {
	// Load configuration from environment variables
	proxy := loadEnvConfig()

	// Get listen port from environment or use default
	listenPortStr := os.Getenv("LISTEN_PORT")
	listenPort, err := strconv.Atoi(listenPortStr)
	if err != nil || listenPort == 0 {
		listenPort = 9001 // Default port
	}

	// Start proxy
	if err := proxy.Start(listenPort); err != nil {
		log.Fatal(err)
	}
}
