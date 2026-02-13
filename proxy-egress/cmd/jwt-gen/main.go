// JWT token generator for testing MarchProxy authentication
package main

import (
	"fmt"
	"os"

	"marchproxy-egress/internal/auth"
	"marchproxy-egress/internal/manager"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s <service_id> <service_name> <jwt_secret>\n", os.Args[0])
		fmt.Printf("Example: %s 1 \"Web Service\" \"my-secret-key\"\n", os.Args[0])
		os.Exit(1)
	}
	
	var serviceID int
	if _, err := fmt.Sscanf(os.Args[1], "%d", &serviceID); err != nil {
		fmt.Printf("Invalid service ID: %v\n", err)
		os.Exit(1)
	}
	
	serviceName := os.Args[2]
	jwtSecret := os.Args[3]
	
	// Create a test service configuration
	service := manager.Service{
		ID:         serviceID,
		Name:       serviceName,
		IPFQDN:     "localhost",
		AuthType:   "jwt",
		JWTSecret:  jwtSecret,
		JWTExpiry:  3600, // 1 hour
	}
	
	// Create authenticator with test service
	authenticator := auth.NewAuthenticator([]manager.Service{service})
	
	// Generate JWT token
	token, err := authenticator.GenerateJWTToken(serviceID)
	if err != nil {
		fmt.Printf("Failed to generate JWT token: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Generated JWT token for service %d (%s):\n", serviceID, serviceName)
	fmt.Printf("%s\n", token)
	fmt.Printf("\nTo test with proxy:\n")
	fmt.Printf("./test-client localhost:8080 %d \"%s\"\n", serviceID, token)
	
	// Test validation
	fmt.Printf("\nValidating token...\n")
	if err := authenticator.AuthenticateService(serviceID, token); err != nil {
		fmt.Printf("Token validation failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Token validation successful!\n")
}