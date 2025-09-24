// MarchProxy - High-performance proxy server with eBPF acceleration
// Main entry point for the proxy application
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"marchproxy-egress/internal/auth"
	"marchproxy-egress/internal/config"
	"marchproxy-egress/internal/ebpf"
	"marchproxy-egress/internal/manager"
	mtls "marchproxy-egress/internal/tls"
	"github.com/spf13/cobra"
)

var (
	version   = "v0.1.1.1757706677" // Updated from .version file
	buildTime = "unknown"
	gitHash   = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "marchproxy-egress",
		Short: "MarchProxy Egress - Enterprise-grade egress proxy server",
		Long: `MarchProxy Egress is a high-performance egress proxy server with eBPF acceleration.
		
Features:
- Layer 4 and Layer 7 protocol support (TCP, UDP, ICMP, HTTP/HTTPS)
- eBPF acceleration for high-performance packet filtering
- JWT and Base64 token authentication
- Enterprise clustering and license validation
- Prometheus metrics and centralized logging
- Optional network acceleration (DPDK, XDP, SR-IOV)`,
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitHash),
		Run:     runProxy,
	}

	// Add command line flags
	rootCmd.Flags().StringP("config", "c", "", "Configuration file path")
	rootCmd.Flags().StringP("manager-url", "m", "", "Manager server URL")
	rootCmd.Flags().StringP("cluster-api-key", "k", "", "Cluster API key")
	rootCmd.Flags().StringP("listen-port", "p", "8080", "Proxy listen port")
	rootCmd.Flags().StringP("admin-port", "a", "8081", "Admin/metrics port")
	rootCmd.Flags().StringP("log-level", "l", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
	rootCmd.Flags().BoolP("enable-ebpf", "e", true, "Enable eBPF acceleration")
	rootCmd.Flags().BoolP("enable-metrics", "", true, "Enable Prometheus metrics")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runProxy(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.Load(cmd)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Starting MarchProxy Egress %s\n", version)
	fmt.Printf("Manager URL: %s\n", cfg.ManagerURL)
	fmt.Printf("Listen Port: %d\n", cfg.ListenPort)
	fmt.Printf("Admin Port: %d\n", cfg.AdminPort)
	fmt.Printf("Log Level: %s\n", cfg.LogLevel)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize manager client for configuration and registration
	managerClient := manager.NewClient(cfg)

	// Check license status first
	licenseStatus, err := managerClient.GetLicenseStatus()
	if err != nil {
		fmt.Printf("Warning: Failed to check license status: %v\n", err)
	} else {
		fmt.Printf("License: %s (%s) - Proxies: %d/%d\n", 
			licenseStatus.Edition, 
			map[bool]string{true: "Valid", false: "Invalid"}[licenseStatus.Valid],
			licenseStatus.CurrentProxies,
			licenseStatus.MaxProxies)
		
		if !licenseStatus.CanRegister {
			fmt.Printf("Error: Cannot register - proxy limit reached or license invalid\n")
			os.Exit(1)
		}
	}

	// Register proxy with manager
	fmt.Printf("Registering with manager...\n")
	if err := managerClient.Register(cfg); err != nil {
		fmt.Printf("Failed to register with manager: %v\n", err)
		os.Exit(1)
	}

	// Get initial configuration
	initialConfig, err := managerClient.GetConfig()
	if err != nil {
		fmt.Printf("Failed to get initial configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded configuration - Services: %d, Mappings: %d\n",
		len(initialConfig.Services), len(initialConfig.Mappings))

	// Initialize mTLS manager if enabled
	var mtlsManager *mtls.MTLSManager
	if cfg.IsMTLSEnabled() {
		fmt.Printf("mTLS enabled - initializing certificate management\n")
		var err error
		mtlsManager, err = mtls.NewMTLSManager(cfg)
		if err != nil {
			fmt.Printf("Failed to initialize mTLS manager: %v\n", err)
			os.Exit(1)
		}

		// Validate mTLS configuration
		if err := mtlsManager.ValidateConfiguration(); err != nil {
			fmt.Printf("mTLS configuration validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("mTLS configuration validated successfully\n")

		// Print certificate information
		certInfo := mtlsManager.GetCertificateInfo()
		if serverCert, ok := certInfo["server_cert"].(map[string]interface{}); ok {
			fmt.Printf("Server certificate: %s\n", serverCert["subject"])
			fmt.Printf("Certificate validity: %s to %s\n", serverCert["not_before"], serverCert["not_after"])
		}
	}

	// Initialize authenticator and metrics
	authenticator := auth.NewAuthenticator(initialConfig.Services)
	metrics := &ProxyMetrics{}

	// Initialize eBPF manager
	ebpfManager := ebpf.NewManager(cfg.EnableEBPF)
	if cfg.EnableEBPF {
		fmt.Printf("eBPF acceleration enabled\n")
		if err := ebpfManager.LoadProgram(""); err != nil {
			fmt.Printf("Warning: Failed to load eBPF program: %v\n", err)
			fmt.Printf("Continuing with userspace-only mode\n")
		} else {
			// Sync initial configuration
			ebpfManager.UpdateServices(initialConfig.Services)
			ebpfManager.UpdateMappings(initialConfig.Mappings)
		}
	}
	
	// Initialize TCP proxy server
	fmt.Printf("Starting TCP proxy server on port %d...\n", cfg.ListenPort)
	tcpProxyServer := &TCPProxy{
		config:        cfg,
		clusterConfig: initialConfig,
		managerClient: managerClient,
		authenticator: authenticator,
		metrics:       metrics,
		ebpfManager:   ebpfManager,
		mtlsManager:   mtlsManager,
	}
	
	// Initialize UDP proxy server
	fmt.Printf("Starting UDP proxy server on port %d...\n", cfg.ListenPort+1000) // UDP on different port
	udpProxyServer := &UDPProxy{
		config:        cfg,
		clusterConfig: initialConfig,
		managerClient: managerClient,
		authenticator: authenticator,
		metrics:       metrics,
		ebpfManager:   ebpfManager,
		mtlsManager:   mtlsManager,
	}

	// Start configuration refresh loop
	go managerClient.StartConfigRefresh(ctx, cfg, func(config *manager.ClusterConfig) {
		fmt.Printf("Configuration updated - Version: %s\n", config.Version)
		tcpProxyServer.updateConfiguration(config)
		udpProxyServer.updateConfiguration(config)
		
		// Update eBPF maps
		if ebpfManager.IsEnabled() {
			ebpfManager.UpdateServices(config.Services)
			ebpfManager.UpdateMappings(config.Mappings)
		}
	})

	// Start heartbeat loop
	go managerClient.StartHeartbeat(ctx, cfg, func() manager.SystemStats {
		return manager.GetSystemStats()
		// TODO: Add actual connection counts and bytes transferred from proxy server
	})

	// Start TCP proxy server in goroutine
	go func() {
		if err := tcpProxyServer.Start(ctx); err != nil {
			fmt.Printf("TCP proxy server failed: %v\n", err)
			cancel()
		}
	}()
	
	// Start UDP proxy server in goroutine
	go func() {
		if err := udpProxyServer.Start(ctx); err != nil {
			fmt.Printf("UDP proxy server failed: %v\n", err)
			cancel()
		}
	}()

	// Start admin server for health checks and metrics
	if cfg.EnableMetrics {
		go func() {
			if err := startAdminServer(cfg.AdminPort, metrics, ebpfManager, mtlsManager); err != nil {
				fmt.Printf("Failed to start admin server: %v\n", err)
			}
		}()
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal %s, shutting down\n", sig)
	case <-ctx.Done():
		fmt.Printf("Context cancelled, shutting down\n")
	}

	// Graceful shutdown
	fmt.Printf("Starting graceful shutdown...\n")

	// Shutdown proxy servers
	if tcpProxyServer != nil {
		tcpProxyServer.Stop()
	}
	if udpProxyServer != nil {
		udpProxyServer.Stop()
	}

	// Cleanup eBPF resources
	if ebpfManager != nil && ebpfManager.IsEnabled() {
		if err := ebpfManager.Cleanup(); err != nil {
			fmt.Printf("Warning: eBPF cleanup error: %v\n", err)
		}
	}

	fmt.Printf("MarchProxy shutdown complete\n")
}

// ProxyMetrics holds metrics for the proxy servers
type ProxyMetrics struct {
	TCPConnections    int64
	UDPPackets        int64
	BytesTransferred  int64
	AuthSuccesses     int64
	AuthFailures      int64
	ActiveConnections int64
	mu                sync.RWMutex
}

// TCPProxy implements a basic TCP proxy server
type TCPProxy struct {
	config        *config.Config
	clusterConfig *manager.ClusterConfig
	managerClient *manager.Client
	authenticator *auth.Authenticator
	metrics       *ProxyMetrics
	ebpfManager   *ebpf.Manager
	mtlsManager   *mtls.MTLSManager
	listener      net.Listener
	wg            sync.WaitGroup
	stopping      bool
	mu            sync.RWMutex
}

// Start starts the TCP proxy server
func (p *TCPProxy) Start(ctx context.Context) error {
	var listener net.Listener
	var err error

	// Create listener with or without TLS based on mTLS configuration
	if p.config.IsMTLSEnabled() && p.mtlsManager != nil {
		// Create TLS listener
		tlsConfig := p.mtlsManager.GetTLSConfig()
		listener, err = tls.Listen("tcp", p.config.GetListenAddress(), tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to create TLS listener on %s: %w", p.config.GetListenAddress(), err)
		}
		fmt.Printf("TCP proxy listening on %s with mTLS enabled\n", p.config.GetListenAddress())
	} else {
		// Create regular TCP listener
		listener, err = net.Listen("tcp", p.config.GetListenAddress())
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", p.config.GetListenAddress(), err)
		}
		fmt.Printf("TCP proxy listening on %s\n", p.config.GetListenAddress())
	}

	p.listener = listener
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			p.mu.RLock()
			stopping := p.stopping
			p.mu.RUnlock()
			
			if stopping {
				break
			}
			
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		
		p.wg.Add(1)
		go p.handleConnection(conn)
	}
	
	return nil
}

// Stop stops the TCP proxy server
func (p *TCPProxy) Stop() {
	p.mu.Lock()
	p.stopping = true
	p.mu.Unlock()
	
	if p.listener != nil {
		p.listener.Close()
	}
	
	p.wg.Wait()
}

// handleConnection handles a single TCP connection
func (p *TCPProxy) handleConnection(clientConn net.Conn) {
	defer p.wg.Done()
	defer clientConn.Close()
	
	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.TCPConnections++
	p.metrics.ActiveConnections++
	p.metrics.mu.Unlock()
	
	defer func() {
		p.metrics.mu.Lock()
		p.metrics.ActiveConnections--
		p.metrics.mu.Unlock()
	}()
	
	fmt.Printf("New connection from %s\n", clientConn.RemoteAddr())

	// Log mTLS connection details if enabled
	if p.config.IsMTLSEnabled() {
		if tlsConn, ok := clientConn.(*tls.Conn); ok {
			// Perform TLS handshake to get certificate info
			if err := tlsConn.Handshake(); err != nil {
				fmt.Printf("TLS handshake failed for %s: %v\n", clientConn.RemoteAddr(), err)
				return
			}

			connectionState := tlsConn.ConnectionState()
			fmt.Printf("mTLS connection established with %s (TLS %s, cipher %s)\n",
				clientConn.RemoteAddr(),
				fmt.Sprintf("1.%d", connectionState.Version&0xff),
				tls.CipherSuiteName(connectionState.CipherSuite))

			if len(connectionState.PeerCertificates) > 0 {
				clientCert := connectionState.PeerCertificates[0]
				fmt.Printf("Client certificate: CN=%s, Serial=%s\n",
					clientCert.Subject.CommonName, clientCert.SerialNumber.String())
			}
		}
	}

	// Check if eBPF should handle this connection
	if p.ebpfManager != nil && p.ebpfManager.IsEnabled() {
		// Parse connection details for eBPF check
		srcIP := ipStringToUint32(getIPFromAddr(clientConn.RemoteAddr()))
		dstIP := ipStringToUint32(getIPFromAddr(clientConn.LocalAddr()))
		srcPort := uint16(getPortFromAddr(clientConn.RemoteAddr()))
		dstPort := uint16(getPortFromAddr(clientConn.LocalAddr()))
		
		// Check if this connection should be handled in userspace
		if !p.ebpfManager.ShouldFallbackToUserspace(srcIP, dstIP, srcPort, dstPort, 6) { // TCP = 6
			// eBPF should handle this - close connection as eBPF will forward
			fmt.Printf("eBPF handling connection from %s\n", clientConn.RemoteAddr())
			return
		}
		fmt.Printf("eBPF fallback: handling in userspace %s\n", clientConn.RemoteAddr())
	}
	
	// Find a matching mapping for this connection
	mapping := p.findMatchingMapping()
	if mapping == nil {
		fmt.Printf("No mapping found for connection from %s\n", clientConn.RemoteAddr())
		return
	}
	
	// Check if authentication is required for this mapping
	if mapping.AuthRequired {
		if err := p.handleAuthentication(clientConn, mapping); err != nil {
			fmt.Printf("Authentication failed for %s: %v\n", clientConn.RemoteAddr(), err)
			return
		}
	}
	
	// Find destination service
	destService := p.findDestinationService(mapping)
	if destService == nil {
		fmt.Printf("No destination service found for mapping %s\n", mapping.Name)
		return
	}
	
	// Connect to destination - use mapping ports or default to 80
	destPort := p.getDestinationPort(mapping)
	destAddr := fmt.Sprintf("%s:%d", destService.IPFQDN, destPort)

	var destConn net.Conn
	// Use mTLS for outbound connections if configured
	if p.config.IsMTLSEnabled() && p.mtlsManager != nil {
		// Create mTLS client for outbound connection
		httpClient, err := p.mtlsManager.CreateHTTPClient()
		if err != nil {
			fmt.Printf("Failed to create mTLS client for %s: %v\n", destAddr, err)
			return
		}

		// For TCP proxy, we need to establish a direct TLS connection
		if httpClient.Transport != nil {
			if transport, ok := httpClient.Transport.(*http.Transport); ok && transport.TLSClientConfig != nil {
				destConn, err = tls.Dial("tcp", destAddr, transport.TLSClientConfig)
				if err != nil {
					fmt.Printf("Failed to establish mTLS connection to %s: %v\n", destAddr, err)
					return
				}
				fmt.Printf("mTLS connection established to destination %s\n", destAddr)
			} else {
				// Fallback to regular connection
				destConn, err = net.Dial("tcp", destAddr)
				if err != nil {
					fmt.Printf("Failed to connect to destination %s: %v\n", destAddr, err)
					return
				}
			}
		} else {
			// Fallback to regular connection
			destConn, err = net.Dial("tcp", destAddr)
			if err != nil {
				fmt.Printf("Failed to connect to destination %s: %v\n", destAddr, err)
				return
			}
		}
	} else {
		// Regular TCP connection
		destConn, err = net.Dial("tcp", destAddr)
		if err != nil {
			fmt.Printf("Failed to connect to destination %s: %v\n", destAddr, err)
			return
		}
	}
	defer destConn.Close()
	
	fmt.Printf("Proxying connection from %s to %s (%s)\n", 
		clientConn.RemoteAddr(), destAddr, destService.Name)
	
	// Start bidirectional forwarding
	errChan := make(chan error, 2)
	
	// Forward client -> server
	go func() {
		_, err := io.Copy(destConn, clientConn)
		errChan <- err
	}()
	
	// Forward server -> client
	go func() {
		_, err := io.Copy(clientConn, destConn)
		errChan <- err
	}()
	
	// Wait for either direction to close
	err = <-errChan
	if err != nil && err != io.EOF {
		fmt.Printf("Proxy error: %v\n", err)
	}
	
	fmt.Printf("Connection from %s to %s closed\n", clientConn.RemoteAddr(), destAddr)
}

// handleAuthentication performs authentication for a connection
func (p *TCPProxy) handleAuthentication(conn net.Conn, mapping *manager.Mapping) error {
	// Send authentication challenge
	authMsg := "MARCHPROXY_AUTH\nPlease provide authentication in format:\nSERVICE_ID:TOKEN\n"
	if _, err := conn.Write([]byte(authMsg)); err != nil {
		return fmt.Errorf("failed to send auth challenge: %w", err)
	}
	
	// Read authentication response
	reader := bufio.NewReader(conn)
	responseLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}
	response := strings.TrimSpace(responseLine)
	
	// Parse service ID and token
	parts := strings.SplitN(response, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid auth format, expected SERVICE_ID:TOKEN")
	}
	
	var serviceID int
	if _, err := fmt.Sscanf(parts[0], "%d", &serviceID); err != nil {
		return fmt.Errorf("invalid service ID: %w", err)
	}
	
	token := parts[1]
	
	// Verify service ID is allowed for this mapping
	allowed := false
	for _, allowedServiceID := range mapping.SourceServices {
		if allowedServiceID == serviceID {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return fmt.Errorf("service %d not allowed for mapping %s", serviceID, mapping.Name)
	}
	
	// Authenticate the service
	if err := p.authenticator.AuthenticateService(serviceID, token); err != nil {
		p.metrics.mu.Lock()
		p.metrics.AuthFailures++
		p.metrics.mu.Unlock()
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	p.metrics.mu.Lock()
	p.metrics.AuthSuccesses++
	p.metrics.mu.Unlock()
	
	// Send success response
	if _, err := conn.Write([]byte("AUTH_OK\n")); err != nil {
		return fmt.Errorf("failed to send auth success: %w", err)
	}
	
	fmt.Printf("Authentication successful for service %d from %s\n", serviceID, conn.RemoteAddr())
	return nil
}

// findMatchingMapping finds the first mapping that matches this connection
func (p *TCPProxy) findMatchingMapping() *manager.Mapping {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.clusterConfig == nil {
		return nil
	}
	
	// For now, return the first mapping with TCP protocol support
	for _, mapping := range p.clusterConfig.Mappings {
		for _, protocol := range mapping.Protocols {
			if protocol == "tcp" {
				return &mapping
			}
		}
	}
	
	return nil
}

// findDestinationService finds a destination service for the mapping
func (p *TCPProxy) findDestinationService(mapping *manager.Mapping) *manager.Service {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.clusterConfig == nil {
		return nil
	}
	
	// Find first available destination service
	for _, serviceID := range mapping.DestServices {
		for _, service := range p.clusterConfig.Services {
			if service.ID == serviceID {
				return &service
			}
		}
	}
	
	return nil
}

// getDestinationPort returns the destination port from mapping or defaults to 80
func (p *TCPProxy) getDestinationPort(mapping *manager.Mapping) int {
	// Parse mapping ports - can be single port, range, or list
	ports := mapping.Ports
	if ports == "" {
		return 80 // Default to HTTP port
	}
	
	// For now, just parse single port or take first port from list
	var port int
	if _, err := fmt.Sscanf(ports, "%d", &port); err == nil {
		return port
	}
	
	// If parsing fails, check for comma-separated list
	if parts := strings.Split(ports, ","); len(parts) > 0 {
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &port); err == nil {
			return port
		}
	}
	
	// Default to port 80 if all parsing fails
	return 80
}

// updateConfiguration updates the proxy's cluster configuration
func (p *TCPProxy) updateConfiguration(config *manager.ClusterConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.clusterConfig = config
	p.authenticator.UpdateServices(config.Services)
	
	fmt.Printf("Proxy configuration updated - Services: %d, Mappings: %d\n", 
		len(config.Services), len(config.Mappings))
}

// UDPProxy implements a UDP proxy server
type UDPProxy struct {
	config        *config.Config
	clusterConfig *manager.ClusterConfig
	managerClient *manager.Client
	authenticator *auth.Authenticator
	metrics       *ProxyMetrics
	ebpfManager   *ebpf.Manager
	mtlsManager   *mtls.MTLSManager
	conn          *net.UDPConn
	stopping      bool
	mu            sync.RWMutex
}

// Start starts the UDP proxy server
func (p *UDPProxy) Start(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", p.config.ListenPort+1000))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %w", udpAddr, err)
	}
	
	p.conn = conn
	fmt.Printf("UDP proxy listening on %s\n", udpAddr)
	
	buffer := make([]byte, 4096)
	for {
		p.mu.RLock()
		stopping := p.stopping
		p.mu.RUnlock()
		
		if stopping {
			break
		}
		
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if stopping {
				break
			}
			fmt.Printf("UDP read error: %v\n", err)
			continue
		}
		
		// Handle UDP packet in goroutine for concurrency
		go p.handleUDPPacket(buffer[:n], clientAddr)
	}
	
	return nil
}

// Stop stops the UDP proxy server
func (p *UDPProxy) Stop() {
	p.mu.Lock()
	p.stopping = true
	p.mu.Unlock()
	
	if p.conn != nil {
		p.conn.Close()
	}
}

// handleUDPPacket handles a single UDP packet
func (p *UDPProxy) handleUDPPacket(data []byte, clientAddr *net.UDPAddr) {
	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.UDPPackets++
	p.metrics.BytesTransferred += int64(len(data))
	p.metrics.mu.Unlock()
	
	fmt.Printf("UDP packet from %s, size: %d bytes\n", clientAddr, len(data))
	
	// Find a matching mapping for UDP traffic
	mapping := p.findMatchingUDPMapping()
	if mapping == nil {
		fmt.Printf("No UDP mapping found for packet from %s\n", clientAddr)
		return
	}
	
	// Find destination service
	destService := p.findDestinationService(mapping)
	if destService == nil {
		fmt.Printf("No destination service found for UDP mapping %s\n", mapping.Name)
		return
	}
	
	// For UDP, we don't have persistent connections, so we forward each packet individually
	destPort := p.getDestinationPort(mapping)
	destAddr := fmt.Sprintf("%s:%d", destService.IPFQDN, destPort)
	destUDPAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		fmt.Printf("Failed to resolve destination UDP address %s: %v\n", destAddr, err)
		return
	}
	
	// Create a connection to destination
	destConn, err := net.DialUDP("udp", nil, destUDPAddr)
	if err != nil {
		fmt.Printf("Failed to connect to UDP destination %s: %v\n", destAddr, err)
		return
	}
	defer destConn.Close()
	
	// Forward the packet
	_, err = destConn.Write(data)
	if err != nil {
		fmt.Printf("Failed to forward UDP packet to %s: %v\n", destAddr, err)
		return
	}
	
	// Read response
	responseBuffer := make([]byte, 4096)
	destConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := destConn.Read(responseBuffer)
	if err != nil {
		fmt.Printf("Failed to read UDP response from %s: %v\n", destAddr, err)
		return
	}
	
	// Send response back to client
	_, err = p.conn.WriteToUDP(responseBuffer[:n], clientAddr)
	if err != nil {
		fmt.Printf("Failed to send UDP response to %s: %v\n", clientAddr, err)
		return
	}
	
	// Update response metrics
	p.metrics.mu.Lock()
	p.metrics.BytesTransferred += int64(n)
	p.metrics.mu.Unlock()
	
	fmt.Printf("UDP packet forwarded: %s -> %s -> %s\n", clientAddr, destAddr, clientAddr)
}

// findMatchingUDPMapping finds the first mapping that supports UDP
func (p *UDPProxy) findMatchingUDPMapping() *manager.Mapping {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.clusterConfig == nil {
		return nil
	}
	
	// For now, return the first mapping with UDP protocol support
	for _, mapping := range p.clusterConfig.Mappings {
		for _, protocol := range mapping.Protocols {
			if protocol == "udp" {
				return &mapping
			}
		}
	}
	
	return nil
}

// findDestinationService finds a destination service for the mapping (shared with TCP)
func (p *UDPProxy) findDestinationService(mapping *manager.Mapping) *manager.Service {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.clusterConfig == nil {
		return nil
	}
	
	// Find first available destination service
	for _, serviceID := range mapping.DestServices {
		for _, service := range p.clusterConfig.Services {
			if service.ID == serviceID {
				return &service
			}
		}
	}
	
	return nil
}

// getDestinationPort returns the destination port from mapping or defaults to 53 for UDP
func (p *UDPProxy) getDestinationPort(mapping *manager.Mapping) int {
	// Parse mapping ports - can be single port, range, or list
	ports := mapping.Ports
	if ports == "" {
		return 53 // Default to DNS port for UDP
	}
	
	// For now, just parse single port or take first port from list
	var port int
	if _, err := fmt.Sscanf(ports, "%d", &port); err == nil {
		return port
	}
	
	// If parsing fails, check for comma-separated list
	if parts := strings.Split(ports, ","); len(parts) > 0 {
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &port); err == nil {
			return port
		}
	}
	
	// Default to port 53 if all parsing fails
	return 53
}

// updateConfiguration updates the proxy's cluster configuration
func (p *UDPProxy) updateConfiguration(config *manager.ClusterConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.clusterConfig = config
	p.authenticator.UpdateServices(config.Services)
}

// startAdminServer starts the admin/metrics HTTP server
func startAdminServer(port int, metrics *ProxyMetrics, ebpfMgr *ebpf.Manager, mtlsMgr *mtls.MTLSManager) error {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		mtlsStatus := "disabled"
		if mtlsMgr != nil {
			certInfo := mtlsMgr.GetCertificateInfo()
			if enabled, ok := certInfo["enabled"].(bool); ok && enabled {
				mtlsStatus = "enabled"
			}
		}

		fmt.Fprintf(w, `{"status":"healthy","version":"%s","mtls":"%s"}`, version, mtlsStatus)
	})
	
	// Comprehensive metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.mu.RLock()
		tcpConnections := metrics.TCPConnections
		udpPackets := metrics.UDPPackets
		bytesTransferred := metrics.BytesTransferred
		authSuccesses := metrics.AuthSuccesses
		authFailures := metrics.AuthFailures
		activeConnections := metrics.ActiveConnections
		metrics.mu.RUnlock()
		
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		
		// TCP connection metrics
		fmt.Fprintf(w, "# HELP marchproxy_tcp_connections_total Total number of TCP connections\n")
		fmt.Fprintf(w, "# TYPE marchproxy_tcp_connections_total counter\n")
		fmt.Fprintf(w, "marchproxy_tcp_connections_total %d\n", tcpConnections)
		
		// UDP packet metrics
		fmt.Fprintf(w, "# HELP marchproxy_udp_packets_total Total number of UDP packets\n")
		fmt.Fprintf(w, "# TYPE marchproxy_udp_packets_total counter\n")
		fmt.Fprintf(w, "marchproxy_udp_packets_total %d\n", udpPackets)
		
		// Bytes transferred metrics
		fmt.Fprintf(w, "# HELP marchproxy_bytes_transferred_total Total bytes transferred\n")
		fmt.Fprintf(w, "# TYPE marchproxy_bytes_transferred_total counter\n")
		fmt.Fprintf(w, "marchproxy_bytes_transferred_total %d\n", bytesTransferred)
		
		// Authentication metrics
		fmt.Fprintf(w, "# HELP marchproxy_auth_successes_total Total successful authentications\n")
		fmt.Fprintf(w, "# TYPE marchproxy_auth_successes_total counter\n")
		fmt.Fprintf(w, "marchproxy_auth_successes_total %d\n", authSuccesses)
		
		fmt.Fprintf(w, "# HELP marchproxy_auth_failures_total Total failed authentications\n")
		fmt.Fprintf(w, "# TYPE marchproxy_auth_failures_total counter\n")
		fmt.Fprintf(w, "marchproxy_auth_failures_total %d\n", authFailures)
		
		// Active connections gauge
		fmt.Fprintf(w, "# HELP marchproxy_active_connections Current number of active connections\n")
		fmt.Fprintf(w, "# TYPE marchproxy_active_connections gauge\n")
		fmt.Fprintf(w, "marchproxy_active_connections %d\n", activeConnections)
		
		// Version information
		fmt.Fprintf(w, "# HELP marchproxy_version_info Version information\n")
		fmt.Fprintf(w, "# TYPE marchproxy_version_info gauge\n")
		fmt.Fprintf(w, `marchproxy_version_info{version="%s"} 1`+"\n", version)

		// mTLS metrics
		if mtlsMgr != nil {
			certInfo := mtlsMgr.GetCertificateInfo()
			mtlsEnabled := 0
			if enabled, ok := certInfo["enabled"].(bool); ok && enabled {
				mtlsEnabled = 1
			}

			fmt.Fprintf(w, "# HELP marchproxy_mtls_enabled Whether mTLS is enabled\n")
			fmt.Fprintf(w, "# TYPE marchproxy_mtls_enabled gauge\n")
			fmt.Fprintf(w, "marchproxy_mtls_enabled %d\n", mtlsEnabled)

			if mtlsEnabled == 1 {
				requireClientCert := 0
				verifyClientCert := 0

				if require, ok := certInfo["require_client_cert"].(bool); ok && require {
					requireClientCert = 1
				}
				if verify, ok := certInfo["verify_client_cert"].(bool); ok && verify {
					verifyClientCert = 1
				}

				fmt.Fprintf(w, "# HELP marchproxy_mtls_require_client_cert Whether client certificates are required\n")
				fmt.Fprintf(w, "# TYPE marchproxy_mtls_require_client_cert gauge\n")
				fmt.Fprintf(w, "marchproxy_mtls_require_client_cert %d\n", requireClientCert)

				fmt.Fprintf(w, "# HELP marchproxy_mtls_verify_client_cert Whether client certificates are verified\n")
				fmt.Fprintf(w, "# TYPE marchproxy_mtls_verify_client_cert gauge\n")
				fmt.Fprintf(w, "marchproxy_mtls_verify_client_cert %d\n", verifyClientCert)
			}
		}

		// eBPF metrics
		if ebpfMgr != nil && ebpfMgr.IsEnabled() {
			ebpfProxyStats, ebpfStats := ebpfMgr.GetStats()
			
			fmt.Fprintf(w, "# HELP marchproxy_ebpf_enabled Whether eBPF acceleration is enabled\n")
			fmt.Fprintf(w, "# TYPE marchproxy_ebpf_enabled gauge\n")
			fmt.Fprintf(w, "marchproxy_ebpf_enabled %d\n", map[bool]int{true: 1, false: 0}[ebpfStats.ProgramLoaded])
			
			fmt.Fprintf(w, "# HELP marchproxy_ebpf_total_packets Total packets processed by eBPF\n")
			fmt.Fprintf(w, "# TYPE marchproxy_ebpf_total_packets counter\n")
			fmt.Fprintf(w, "marchproxy_ebpf_total_packets %d\n", ebpfProxyStats.TotalPackets)
			
			fmt.Fprintf(w, "# HELP marchproxy_ebpf_dropped_packets Packets dropped by eBPF\n")
			fmt.Fprintf(w, "# TYPE marchproxy_ebpf_dropped_packets counter\n")
			fmt.Fprintf(w, "marchproxy_ebpf_dropped_packets %d\n", ebpfProxyStats.DroppedPackets)
			
			fmt.Fprintf(w, "# HELP marchproxy_ebpf_forwarded_packets Packets forwarded by eBPF\n")
			fmt.Fprintf(w, "# TYPE marchproxy_ebpf_forwarded_packets counter\n")
			fmt.Fprintf(w, "marchproxy_ebpf_forwarded_packets %d\n", ebpfProxyStats.ForwardedPackets)
			
			fmt.Fprintf(w, "# HELP marchproxy_ebpf_userspace_fallback Packets sent to userspace\n")
			fmt.Fprintf(w, "# TYPE marchproxy_ebpf_userspace_fallback counter\n")
			fmt.Fprintf(w, "marchproxy_ebpf_userspace_fallback %d\n", ebpfProxyStats.FallbackToUserspace)
			
			fmt.Fprintf(w, "# HELP marchproxy_ebpf_map_sync_errors eBPF map synchronization errors\n")
			fmt.Fprintf(w, "# TYPE marchproxy_ebpf_map_sync_errors counter\n")
			fmt.Fprintf(w, "marchproxy_ebpf_map_sync_errors %d\n", ebpfStats.MapSyncErrors)
		}
	})
	
	// Stats endpoint for easy debugging
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		metrics.mu.RLock()
		tcpConnections := metrics.TCPConnections
		udpPackets := metrics.UDPPackets
		bytesTransferred := metrics.BytesTransferred
		authSuccesses := metrics.AuthSuccesses
		authFailures := metrics.AuthFailures
		activeConnections := metrics.ActiveConnections
		metrics.mu.RUnlock()
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		ebpfSection := ""
		if ebpfMgr != nil && ebpfMgr.IsEnabled() {
			ebpfProxyStats, ebpfStats := ebpfMgr.GetStats()
			ebpfSection = fmt.Sprintf(`,
	"ebpf": {
		"enabled": %t,
		"program_loaded": %t,
		"total_packets": %d,
		"dropped_packets": %d,
		"forwarded_packets": %d,
		"userspace_fallback": %d,
		"map_sync_errors": %d,
		"attached_interfaces": %d
	}`, ebpfMgr.IsEnabled(), ebpfStats.ProgramLoaded, ebpfProxyStats.TotalPackets,
				ebpfProxyStats.DroppedPackets, ebpfProxyStats.ForwardedPackets,
				ebpfProxyStats.FallbackToUserspace, ebpfStats.MapSyncErrors,
				len(ebpfStats.AttachedInterfaces))
		}
		
		fmt.Fprintf(w, `{
	"version": "%s",
	"tcp_connections": %d,
	"udp_packets": %d,
	"bytes_transferred": %d,
	"auth_successes": %d,
	"auth_failures": %d,
	"active_connections": %d%s
}`, version, tcpConnections, udpPackets, bytesTransferred,
			authSuccesses, authFailures, activeConnections, ebpfSection)
	})
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	fmt.Printf("Admin server listening on :%d\n", port)
	fmt.Printf("Endpoints: /healthz, /metrics, /stats\n")
	return server.ListenAndServe()
}

// Helper functions for network address parsing

// getIPFromAddr extracts IP address string from net.Addr
func getIPFromAddr(addr net.Addr) string {
	switch v := addr.(type) {
	case *net.TCPAddr:
		return v.IP.String()
	case *net.UDPAddr:
		return v.IP.String()
	default:
		// Parse from string representation
		addrStr := addr.String()
		if host, _, err := net.SplitHostPort(addrStr); err == nil {
			return host
		}
		return "127.0.0.1" // Default fallback
	}
}

// getPortFromAddr extracts port number from net.Addr
func getPortFromAddr(addr net.Addr) int {
	switch v := addr.(type) {
	case *net.TCPAddr:
		return v.Port
	case *net.UDPAddr:
		return v.Port
	default:
		// Parse from string representation
		addrStr := addr.String()
		if _, portStr, err := net.SplitHostPort(addrStr); err == nil {
			if port, err := strconv.Atoi(portStr); err == nil {
				return port
			}
		}
		return 0 // Default fallback
	}
}

// ipStringToUint32 converts IP address string to uint32
func ipStringToUint32(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}