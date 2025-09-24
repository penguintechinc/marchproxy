// Simple test client for MarchProxy authentication
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s <proxy_host:port> <service_id> <token>\n", os.Args[0])
		fmt.Printf("Example: %s localhost:8080 1 mytoken123\n", os.Args[0])
		os.Exit(1)
	}
	
	proxyAddr := os.Args[1]
	serviceID := os.Args[2]
	token := os.Args[3]
	
	// Connect to proxy
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		fmt.Printf("Failed to connect to proxy: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	
	fmt.Printf("Connected to proxy at %s\n", proxyAddr)
	
	// Read authentication challenge
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read from proxy: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Proxy: %s", line)
		
		if strings.Contains(line, "SERVICE_ID:TOKEN") {
			break
		}
	}
	
	// Send authentication
	authResponse := fmt.Sprintf("%s:%s\n", serviceID, token)
	fmt.Printf("Sending auth: %s", authResponse)
	
	if _, err := conn.Write([]byte(authResponse)); err != nil {
		fmt.Printf("Failed to send auth: %v\n", err)
		os.Exit(1)
	}
	
	// Read authentication result
	result, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read auth result: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Auth result: %s", result)
	
	if strings.Contains(result, "AUTH_OK") {
		fmt.Printf("Authentication successful! Now connected to backend service.\n")
		
		// Simple echo test
		testMessage := "Hello from test client!\n"
		if _, err := conn.Write([]byte(testMessage)); err != nil {
			fmt.Printf("Failed to send test message: %v\n", err)
			os.Exit(1)
		}
		
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read response: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Backend response: %s", response)
	} else {
		fmt.Printf("Authentication failed\n")
		os.Exit(1)
	}
}