package zerotrust

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Role represents a user role with permissions
type Role struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	Description string   `json:"description"`
}

// RBACEvaluator handles per-request RBAC evaluation
type RBACEvaluator struct {
	mu              sync.RWMutex
	policyEnforcer  *PolicyEnforcer
	roles           map[string]*Role
	userRoles       map[string][]string
	serviceRoles    map[string][]string
	logger          *logrus.Logger
	cacheEnabled    bool
	cacheTTL        time.Duration
	permissionCache *permissionCache
}

// permissionCache caches permission check results
type permissionCache struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// RBACRequest represents an RBAC evaluation request
type RBACRequest struct {
	Subject    string                 `json:"subject"`      // User, service, or certificate CN
	Action     string                 `json:"action"`       // e.g., "read", "write", "execute"
	Resource   string                 `json:"resource"`     // Resource being accessed
	SourceIP   string                 `json:"source_ip"`    // Client IP address
	Attributes map[string]interface{} `json:"attributes"`   // Additional context
	Timestamp  time.Time              `json:"timestamp"`
}

// RBACResponse contains the result of RBAC evaluation
type RBACResponse struct {
	Allowed     bool     `json:"allowed"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Reason      string   `json:"reason"`
}

// NewRBACEvaluator creates a new RBAC evaluator
func NewRBACEvaluator(policyEnforcer *PolicyEnforcer, logger *logrus.Logger) *RBACEvaluator {
	return &RBACEvaluator{
		policyEnforcer:  policyEnforcer,
		roles:           make(map[string]*Role),
		userRoles:       make(map[string][]string),
		serviceRoles:    make(map[string][]string),
		logger:          logger,
		cacheEnabled:    true,
		cacheTTL:        5 * time.Minute,
		permissionCache: &permissionCache{cache: make(map[string]*cacheEntry)},
	}
}

// LoadRoles loads role definitions
func (re *RBACEvaluator) LoadRoles(roles []*Role) {
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, role := range roles {
		re.roles[role.Name] = role
	}

	re.logger.WithField("count", len(roles)).Info("Loaded RBAC roles")
}

// AssignUserRole assigns a role to a user
func (re *RBACEvaluator) AssignUserRole(user string, roleName string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.roles[roleName]; !exists {
		return fmt.Errorf("role %s does not exist", roleName)
	}

	if re.userRoles[user] == nil {
		re.userRoles[user] = []string{}
	}

	// Check if already assigned
	for _, r := range re.userRoles[user] {
		if r == roleName {
			return nil // Already assigned
		}
	}

	re.userRoles[user] = append(re.userRoles[user], roleName)
	re.logger.WithFields(logrus.Fields{
		"user": user,
		"role": roleName,
	}).Info("Assigned role to user")

	return nil
}

// AssignServiceRole assigns a role to a service
func (re *RBACEvaluator) AssignServiceRole(service string, roleName string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.roles[roleName]; !exists {
		return fmt.Errorf("role %s does not exist", roleName)
	}

	if re.serviceRoles[service] == nil {
		re.serviceRoles[service] = []string{}
	}

	// Check if already assigned
	for _, r := range re.serviceRoles[service] {
		if r == roleName {
			return nil // Already assigned
		}
	}

	re.serviceRoles[service] = append(re.serviceRoles[service], roleName)
	re.logger.WithFields(logrus.Fields{
		"service": service,
		"role":    roleName,
	}).Info("Assigned role to service")

	return nil
}

// EvaluateRBAC evaluates an RBAC request
func (re *RBACEvaluator) EvaluateRBAC(ctx context.Context, req *RBACRequest) (*RBACResponse, error) {
	// Check cache first
	if re.cacheEnabled {
		if cached := re.getCached(req); cached != nil {
			return cached, nil
		}
	}

	// Get subject's roles
	roles := re.getSubjectRoles(req.Subject)
	if len(roles) == 0 {
		return &RBACResponse{
			Allowed: false,
			Reason:  fmt.Sprintf("subject %s has no assigned roles", req.Subject),
		}, nil
	}

	// Get all permissions from roles
	permissions := re.getRolePermissions(roles)

	// Check if any permission matches the requested action
	allowed := re.checkPermission(req.Action, permissions)

	// If local check passes, also validate with OPA if available
	if allowed && re.policyEnforcer != nil && re.policyEnforcer.IsEnabled() {
		policyInput := &PolicyInput{
			Service:   req.Subject,
			Action:    req.Action,
			Resource:  req.Resource,
			SourceIP:  req.SourceIP,
			Timestamp: req.Timestamp,
			Metadata:  req.Attributes,
		}

		policyResult, err := re.policyEnforcer.EvaluatePolicy(ctx, "marchproxy/rbac", policyInput)
		if err != nil {
			re.logger.WithError(err).Warn("OPA policy evaluation failed, using local RBAC result")
		} else {
			allowed = policyResult.Allowed
		}
	}

	response := &RBACResponse{
		Allowed:     allowed,
		Roles:       roles,
		Permissions: permissions,
	}

	if allowed {
		response.Reason = "access granted"
	} else {
		response.Reason = fmt.Sprintf("action %s not permitted for subject %s", req.Action, req.Subject)
	}

	// Cache the result
	if re.cacheEnabled {
		re.setCached(req, response)
	}

	return response, nil
}

// getSubjectRoles retrieves all roles for a subject (user or service)
func (re *RBACEvaluator) getSubjectRoles(subject string) []string {
	re.mu.RLock()
	defer re.mu.RUnlock()

	// Check user roles
	if roles, exists := re.userRoles[subject]; exists {
		return roles
	}

	// Check service roles
	if roles, exists := re.serviceRoles[subject]; exists {
		return roles
	}

	return []string{}
}

// getRolePermissions gets all permissions from a list of roles
func (re *RBACEvaluator) getRolePermissions(roleNames []string) []string {
	re.mu.RLock()
	defer re.mu.RUnlock()

	permissionSet := make(map[string]bool)

	for _, roleName := range roleNames {
		if role, exists := re.roles[roleName]; exists {
			for _, perm := range role.Permissions {
				permissionSet[perm] = true
			}
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions
}

// checkPermission checks if an action is allowed by the permissions
func (re *RBACEvaluator) checkPermission(action string, permissions []string) bool {
	for _, perm := range permissions {
		// Exact match
		if perm == action {
			return true
		}
		// Wildcard match (e.g., "admin:*")
		if len(perm) > 0 && perm[len(perm)-1] == '*' {
			prefix := perm[:len(perm)-1]
			if len(action) >= len(prefix) && action[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// getCached retrieves a cached RBAC result
func (re *RBACEvaluator) getCached(req *RBACRequest) *RBACResponse {
	cacheKey := fmt.Sprintf("%s:%s:%s", req.Subject, req.Action, req.Resource)

	re.permissionCache.mu.RLock()
	defer re.permissionCache.mu.RUnlock()

	if entry, exists := re.permissionCache.cache[cacheKey]; exists {
		if time.Now().Before(entry.expiresAt) {
			roles := re.getSubjectRoles(req.Subject)
			permissions := re.getRolePermissions(roles)

			return &RBACResponse{
				Allowed:     entry.allowed,
				Roles:       roles,
				Permissions: permissions,
			}
		}
	}

	return nil
}

// setCached caches an RBAC result
func (re *RBACEvaluator) setCached(req *RBACRequest, response *RBACResponse) {
	cacheKey := fmt.Sprintf("%s:%s:%s", req.Subject, req.Action, req.Resource)

	re.permissionCache.mu.Lock()
	defer re.permissionCache.mu.Unlock()

	re.permissionCache.cache[cacheKey] = &cacheEntry{
		allowed:   response.Allowed,
		expiresAt: time.Now().Add(re.cacheTTL),
	}
}

// ClearCache clears the permission cache
func (re *RBACEvaluator) ClearCache() {
	re.permissionCache.mu.Lock()
	defer re.permissionCache.mu.Unlock()

	re.permissionCache.cache = make(map[string]*cacheEntry)
	re.logger.Info("Cleared RBAC permission cache")
}

// startCacheCleanup starts a goroutine to periodically clean expired cache entries
func (re *RBACEvaluator) startCacheCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			re.cleanExpiredCache()
		}
	}()
}

// cleanExpiredCache removes expired entries from cache
func (re *RBACEvaluator) cleanExpiredCache() {
	re.permissionCache.mu.Lock()
	defer re.permissionCache.mu.Unlock()

	now := time.Now()
	count := 0

	for key, entry := range re.permissionCache.cache {
		if now.After(entry.expiresAt) {
			delete(re.permissionCache.cache, key)
			count++
		}
	}

	if count > 0 {
		re.logger.WithField("count", count).Debug("Cleaned expired RBAC cache entries")
	}
}
