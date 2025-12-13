package grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ModuleClient represents a gRPC client connection to a module container
type ModuleClient struct {
	name       string
	address    string
	port       int
	conn       *grpc.ClientConn
	state      connectivity.State
	lastUsed   time.Time
	mu         sync.RWMutex
	logger     *logrus.Logger
}

// NewModuleClient creates a new module client
func NewModuleClient(name, address string, port int, logger *logrus.Logger) (*ModuleClient, error) {
	if address == "" || port <= 0 {
		return nil, errors.New("invalid address or port")
	}

	return &ModuleClient{
		name:    name,
		address: address,
		port:    port,
		logger:  logger,
	}, nil
}

// Connect establishes gRPC connection to the module
func (mc *ModuleClient) Connect(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.conn != nil {
		state := mc.conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			return nil
		}
		mc.conn.Close()
	}

	target := fmt.Sprintf("%s:%d", mc.address, mc.port)

	// Configure keepalive parameters
	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaParams),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(16 * 1024 * 1024), // 16MB
			grpc.MaxCallSendMsgSize(16 * 1024 * 1024), // 16MB
		),
	}

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, target, opts...)
	if err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"module": mc.name,
			"target": target,
		}).Error("Failed to connect to module")
		return fmt.Errorf("failed to connect to module %s: %w", mc.name, err)
	}

	mc.conn = conn
	mc.state = conn.GetState()
	mc.lastUsed = time.Now()

	mc.logger.WithFields(logrus.Fields{
		"module": mc.name,
		"target": target,
		"state":  mc.state.String(),
	}).Info("Connected to module")

	return nil
}

// GetConnection returns the gRPC connection
func (mc *ModuleClient) GetConnection() (*grpc.ClientConn, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.conn == nil {
		return nil, errors.New("not connected")
	}

	state := mc.conn.GetState()
	if state != connectivity.Ready && state != connectivity.Idle {
		return nil, fmt.Errorf("connection not ready: %s", state.String())
	}

	mc.lastUsed = time.Now()
	return mc.conn, nil
}

// Close closes the gRPC connection
func (mc *ModuleClient) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.conn == nil {
		return nil
	}

	err := mc.conn.Close()
	mc.conn = nil
	mc.state = connectivity.Shutdown

	mc.logger.WithField("module", mc.name).Info("Module client closed")
	return err
}

// IsHealthy checks if the connection is healthy
func (mc *ModuleClient) IsHealthy() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.conn == nil {
		return false
	}

	state := mc.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// GetState returns current connection state
func (mc *ModuleClient) GetState() connectivity.State {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.conn == nil {
		return connectivity.Shutdown
	}

	return mc.conn.GetState()
}

// ClientPool manages a pool of gRPC client connections
type ClientPool struct {
	clients map[string]*ModuleClient
	mu      sync.RWMutex
	logger  *logrus.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewClientPool creates a new client pool
func NewClientPool(logger *logrus.Logger) *ClientPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ClientPool{
		clients: make(map[string]*ModuleClient),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start health check routine
	pool.wg.Add(1)
	go pool.healthCheckLoop()

	return pool
}

// AddClient adds a client to the pool
func (cp *ClientPool) AddClient(name, address string, port int) error {
	client, err := NewModuleClient(name, address, port, cp.logger)
	if err != nil {
		return err
	}

	// Attempt initial connection
	if err := client.Connect(cp.ctx); err != nil {
		cp.logger.WithError(err).Warn("Initial connection failed, will retry")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, exists := cp.clients[name]; exists {
		return fmt.Errorf("client %s already exists", name)
	}

	cp.clients[name] = client

	cp.logger.WithFields(logrus.Fields{
		"name":    name,
		"address": address,
		"port":    port,
	}).Info("Client added to pool")

	return nil
}

// RemoveClient removes a client from the pool
func (cp *ClientPool) RemoveClient(name string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	client, exists := cp.clients[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	if err := client.Close(); err != nil {
		cp.logger.WithError(err).Warn("Error closing client")
	}

	delete(cp.clients, name)

	cp.logger.WithField("name", name).Info("Client removed from pool")
	return nil
}

// GetClient returns a client by name
func (cp *ClientPool) GetClient(name string) (*ModuleClient, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	client, exists := cp.clients[name]
	if !exists {
		return nil, fmt.Errorf("client %s not found", name)
	}

	return client, nil
}

// GetConnection returns a connection for a specific client
func (cp *ClientPool) GetConnection(name string) (*grpc.ClientConn, error) {
	client, err := cp.GetClient(name)
	if err != nil {
		return nil, err
	}

	return client.GetConnection()
}

// healthCheckLoop periodically checks and reconnects unhealthy clients
func (cp *ClientPool) healthCheckLoop() {
	defer cp.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.checkAndReconnect()
		}
	}
}

// checkAndReconnect checks all clients and reconnects if needed
func (cp *ClientPool) checkAndReconnect() {
	cp.mu.RLock()
	clients := make([]*ModuleClient, 0, len(cp.clients))
	for _, client := range cp.clients {
		clients = append(clients, client)
	}
	cp.mu.RUnlock()

	for _, client := range clients {
		if !client.IsHealthy() {
			cp.logger.WithFields(logrus.Fields{
				"module": client.name,
				"state":  client.GetState().String(),
			}).Warn("Unhealthy client detected, attempting reconnect")

			if err := client.Connect(cp.ctx); err != nil {
				cp.logger.WithError(err).WithField("module", client.name).Error("Reconnection failed")
			} else {
				cp.logger.WithField("module", client.name).Info("Client reconnected successfully")
			}
		}
	}
}

// Close closes all clients in the pool
func (cp *ClientPool) Close() error {
	cp.cancel()
	cp.wg.Wait()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	var lastErr error
	for name, client := range cp.clients {
		if err := client.Close(); err != nil {
			cp.logger.WithError(err).WithField("client", name).Error("Error closing client")
			lastErr = err
		}
	}

	cp.clients = make(map[string]*ModuleClient)
	cp.logger.Info("Client pool closed")

	return lastErr
}

// GetStats returns pool statistics
func (cp *ClientPool) GetStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := make(map[string]interface{})
	clientStats := make(map[string]interface{})

	totalClients := len(cp.clients)
	healthyCount := 0

	for name, client := range cp.clients {
		healthy := client.IsHealthy()
		if healthy {
			healthyCount++
		}

		clientStats[name] = map[string]interface{}{
			"address":   client.address,
			"port":      client.port,
			"healthy":   healthy,
			"state":     client.GetState().String(),
			"last_used": client.lastUsed,
		}
	}

	stats["clients"] = clientStats
	stats["total_clients"] = totalClients
	stats["healthy_clients"] = healthyCount

	return stats
}
