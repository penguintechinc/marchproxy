package pool

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Pool manages database connection pooling
type Pool struct {
	pools       map[string]*ProtocolPool
	maxConns    int
	idleTimeout time.Duration
	logger      *logrus.Logger
	mu          sync.RWMutex
}

// ProtocolPool manages connections for a specific protocol
type ProtocolPool struct {
	protocol    string
	connections chan net.Conn
	maxConns    int
	activeConns int
	totalConns  int64
	mu          sync.RWMutex
}

// NewPool creates a new connection pool
func NewPool(maxConns int, logger *logrus.Logger) *Pool {
	return &Pool{
		pools:       make(map[string]*ProtocolPool),
		maxConns:    maxConns,
		idleTimeout: 5 * time.Minute,
		logger:      logger,
	}
}

// Get retrieves a connection from the pool for the specified protocol
func (p *Pool) Get(protocol string) (net.Conn, error) {
	p.mu.RLock()
	protocolPool, exists := p.pools[protocol]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no pool for protocol: %s", protocol)
	}

	// Try to get connection from pool
	select {
	case conn := <-protocolPool.connections:
		// Validate connection is still alive
		if conn != nil {
			return conn, nil
		}
	default:
		// No available connections in pool
	}

	// Create new connection if under max limit
	protocolPool.mu.Lock()
	if protocolPool.activeConns >= protocolPool.maxConns {
		protocolPool.mu.Unlock()
		return nil, fmt.Errorf("connection pool exhausted for protocol: %s", protocol)
	}
	protocolPool.activeConns++
	protocolPool.totalConns++
	protocolPool.mu.Unlock()

	// In a real implementation, this would connect to the actual backend
	// For now, return a placeholder connection
	conn, err := p.createConnection(protocol)
	if err != nil {
		protocolPool.mu.Lock()
		protocolPool.activeConns--
		protocolPool.mu.Unlock()
		return nil, err
	}

	return conn, nil
}

// Put returns a connection to the pool
func (p *Pool) Put(protocol string, conn net.Conn) {
	p.mu.RLock()
	protocolPool, exists := p.pools[protocol]
	p.mu.RUnlock()

	if !exists {
		if conn != nil {
			conn.Close()
		}
		return
	}

	// Try to return connection to pool
	select {
	case protocolPool.connections <- conn:
		// Connection returned to pool
	default:
		// Pool is full, close connection
		if conn != nil {
			conn.Close()
		}
		protocolPool.mu.Lock()
		protocolPool.activeConns--
		protocolPool.mu.Unlock()
	}
}

// CreatePool creates a new pool for a specific protocol
func (p *Pool) CreatePool(protocol string, maxConns int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.pools[protocol]; exists {
		return fmt.Errorf("pool already exists for protocol: %s", protocol)
	}

	protocolPool := &ProtocolPool{
		protocol:    protocol,
		connections: make(chan net.Conn, maxConns),
		maxConns:    maxConns,
	}

	p.pools[protocol] = protocolPool

	p.logger.WithFields(logrus.Fields{
		"protocol":  protocol,
		"max_conns": maxConns,
	}).Info("Protocol pool created")

	return nil
}

// GetStats returns pool statistics
func (p *Pool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	for protocol, protocolPool := range p.pools {
		protocolPool.mu.RLock()
		stats[protocol] = map[string]interface{}{
			"active_conns": protocolPool.activeConns,
			"total_conns":  protocolPool.totalConns,
			"pool_size":    len(protocolPool.connections),
			"max_conns":    protocolPool.maxConns,
		}
		protocolPool.mu.RUnlock()
	}

	return stats
}

// Close closes all pools and connections
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for protocol, protocolPool := range p.pools {
		close(protocolPool.connections)

		// Drain and close all connections
		for conn := range protocolPool.connections {
			if conn != nil {
				conn.Close()
			}
		}

		p.logger.WithField("protocol", protocol).Info("Protocol pool closed")
	}

	p.pools = make(map[string]*ProtocolPool)
	return nil
}

// createConnection creates a new backend connection
// This is a placeholder implementation
func (p *Pool) createConnection(protocol string) (net.Conn, error) {
	// In a real implementation, this would connect to the actual backend
	// based on protocol and configuration
	// For now, we'll create a mock connection that implements net.Conn

	// Create a pair of connected pipes to simulate a connection
	client, server := net.Pipe()

	// Return the client end as the backend connection
	// In production, this would be: net.Dial("tcp", backendAddr)
	go func() {
		// Simulate backend - in production this would be the actual backend
		defer server.Close()
		// Keep the pipe alive
		buf := make([]byte, 1024)
		for {
			_, err := server.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	return client, nil
}

// Cleanup performs periodic cleanup of idle connections
func (p *Pool) Cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		for _, protocolPool := range p.pools {
			// Clean up idle connections
			select {
			case conn := <-protocolPool.connections:
				if conn != nil {
					conn.Close()
				}
				protocolPool.mu.Lock()
				protocolPool.activeConns--
				protocolPool.mu.Unlock()
			default:
				// No connections to clean up
			}
		}
		p.mu.RUnlock()
	}
}
