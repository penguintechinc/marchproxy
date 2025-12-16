package handlers

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"marchproxy-dblb/internal/config"
	"marchproxy-dblb/internal/pool"
	"marchproxy-dblb/internal/security"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// Handler defines the interface for database protocol handlers
type Handler interface {
	Start(ctx context.Context) error
	Stop() error
	GetStats() map[string]interface{}
}

// Manager manages all database protocol handlers
type Manager struct {
	handlers        map[string]Handler
	pool            *pool.Pool
	securityChecker *security.Checker
	config          *config.Config
	logger          *logrus.Logger
	mu              sync.RWMutex
}

// NewManager creates a new handler manager
func NewManager(pool *pool.Pool, securityChecker *security.Checker, cfg *config.Config, logger *logrus.Logger) *Manager {
	return &Manager{
		handlers:        make(map[string]Handler),
		pool:            pool,
		securityChecker: securityChecker,
		config:          cfg,
		logger:          logger,
	}
}

// RegisterHandler registers a database protocol handler
func (m *Manager) RegisterHandler(protocol string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.handlers[protocol]; exists {
		return fmt.Errorf("handler for protocol %s already registered", protocol)
	}

	// Create a generic TCP handler for the protocol
	handler := NewTCPHandler(protocol, port, m.pool, m.securityChecker, m.config, m.logger)
	m.handlers[protocol] = handler

	m.logger.WithFields(logrus.Fields{
		"protocol": protocol,
		"port":     port,
	}).Info("Handler registered")

	return nil
}

// StartAll starts all registered handlers
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for protocol, handler := range m.handlers {
		if err := handler.Start(ctx); err != nil {
			return fmt.Errorf("failed to start handler for %s: %w", protocol, err)
		}
	}

	return nil
}

// StopAll stops all registered handlers
func (m *Manager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastErr error
	for protocol, handler := range m.handlers {
		if err := handler.Stop(); err != nil {
			m.logger.WithError(err).Errorf("Failed to stop handler for %s", protocol)
			lastErr = err
		}
	}

	return lastErr
}

// GetStats returns statistics for all handlers
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	for protocol, handler := range m.handlers {
		stats[protocol] = handler.GetStats()
	}

	return stats
}

// GetHandler returns a specific handler by protocol
func (m *Manager) GetHandler(protocol string) (Handler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handler, exists := m.handlers[protocol]
	return handler, exists
}

// TCPHandler implements a generic TCP proxy handler for database protocols
type TCPHandler struct {
	protocol        string
	port            int
	pool            *pool.Pool
	securityChecker *security.Checker
	config          *config.Config
	logger          *logrus.Logger
	listener        net.Listener
	connLimiter     *rate.Limiter
	queryLimiter    *rate.Limiter
	activeConns     int64
	totalConns      int64
	mu              sync.RWMutex
	running         bool
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewTCPHandler creates a new TCP handler for a database protocol
func NewTCPHandler(protocol string, port int, pool *pool.Pool, securityChecker *security.Checker, cfg *config.Config, logger *logrus.Logger) *TCPHandler {
	return &TCPHandler{
		protocol:        protocol,
		port:            port,
		pool:            pool,
		securityChecker: securityChecker,
		config:          cfg,
		logger:          logger,
		connLimiter:     rate.NewLimiter(rate.Limit(cfg.DefaultConnectionRate), int(cfg.DefaultConnectionRate)),
		queryLimiter:    rate.NewLimiter(rate.Limit(cfg.DefaultQueryRate), int(cfg.DefaultQueryRate)),
	}
}

// Start starts the TCP handler
func (h *TCPHandler) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return fmt.Errorf("handler already running")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", h.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", h.port, err)
	}

	h.listener = listener
	h.ctx, h.cancel = context.WithCancel(ctx)
	h.running = true

	go h.acceptConnections()

	h.logger.WithFields(logrus.Fields{
		"protocol": h.protocol,
		"port":     h.port,
	}).Info("TCP handler started")

	return nil
}

// Stop stops the TCP handler
func (h *TCPHandler) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return nil
	}

	h.logger.WithField("protocol", h.protocol).Info("Stopping TCP handler")

	if h.cancel != nil {
		h.cancel()
	}

	if h.listener != nil {
		h.listener.Close()
	}

	h.running = false
	return nil
}

// GetStats returns handler statistics
func (h *TCPHandler) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"protocol":     h.protocol,
		"port":         h.port,
		"active_conns": h.activeConns,
		"total_conns":  h.totalConns,
		"running":      h.running,
	}
}

// acceptConnections accepts incoming connections
func (h *TCPHandler) acceptConnections() {
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
			conn, err := h.listener.Accept()
			if err != nil {
				if !h.isRunning() {
					return
				}
				h.logger.WithError(err).Error("Failed to accept connection")
				continue
			}

			// Apply rate limiting
			if !h.connLimiter.Allow() {
				h.logger.Warn("Connection rate limit exceeded")
				conn.Close()
				continue
			}

			go h.handleConnection(conn)
		}
	}
}

// handleConnection handles a single database connection
func (h *TCPHandler) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	h.incrementActiveConns()
	defer h.decrementActiveConns()

	h.incrementTotalConns()

	// Get backend connection from pool
	backendConn, err := h.pool.Get(h.protocol)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get backend connection")
		return
	}
	defer h.pool.Put(h.protocol, backendConn)

	// Bidirectional proxy
	errChan := make(chan error, 2)

	// Client to backend
	go func() {
		_, err := io.Copy(backendConn, clientConn)
		errChan <- err
	}()

	// Backend to client
	go func() {
		_, err := io.Copy(clientConn, backendConn)
		errChan <- err
	}()

	// Wait for first error or completion
	<-errChan
}

// isRunning returns whether the handler is running
func (h *TCPHandler) isRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// incrementActiveConns increments active connection counter
func (h *TCPHandler) incrementActiveConns() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeConns++
}

// decrementActiveConns decrements active connection counter
func (h *TCPHandler) decrementActiveConns() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeConns--
}

// incrementTotalConns increments total connection counter
func (h *TCPHandler) incrementTotalConns() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.totalConns++
}
