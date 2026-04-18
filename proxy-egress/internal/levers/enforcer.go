package levers

import (
	"fmt"
	"net"

	"marchproxy-egress/internal/logging"
)

var logEnforcer *logging.LogrusAdapter

func init() {
	var err error
	logEnforcer, err = logging.NewLogrusAdapter("levers-enforcer")
	if err != nil {
		panic(err)
	}
}

// Enforcer applies received rule sets to MarchProxy's enforcement mechanisms.
// It writes to eBPF maps for IP rules and updates Envoy config for domain/routing rules.
// No evaluation logic — it only translates and applies what the controller sent.
type Enforcer struct {
	// ebpfMgr is the eBPF map manager (set to nil if eBPF disabled)
	ebpfMgr EBPFManager
}

// EBPFManager abstracts eBPF map operations so tests can mock them.
type EBPFManager interface {
	BlockCIDR(cidr string) error
	AllowCIDR(cidr string) error
	SetRateLimit(srcIP string, pps int) error
	ClearBlocklist() error
	ClearAllowlist() error
}

func NewEnforcer(ebpfMgr EBPFManager) *Enforcer {
	return &Enforcer{ebpfMgr: ebpfMgr}
}

// Apply applies the full rule set atomically (clear existing, apply new).
func (e *Enforcer) Apply(rules RuleSet) error {
	if e.ebpfMgr != nil {
		if err := e.applyEBPFRules(rules); err != nil {
			return fmt.Errorf("ebpf rules: %w", err)
		}
	} else {
		logEnforcer.Debug("levers: eBPF manager not configured — IP rules logged only")
		for _, cidr := range rules.BlockCIDRs {
			logEnforcer.WithField("cidr", cidr).Debug("levers: would block CIDR")
		}
	}
	// Domain and cluster rules: logged as stubs pending Envoy xDS integration
	for _, domain := range rules.BlockDomains {
		logEnforcer.WithField("domain", domain).Debug("levers: block domain (Envoy xDS TBD)")
	}
	for _, cluster := range rules.RouteClusters {
		logEnforcer.WithFields(map[string]interface{}{
			"cluster":   cluster.Name,
			"endpoints": cluster.Endpoints,
			"lb_policy": cluster.LBPolicy,
		}).Debug("levers: route cluster (Envoy xDS TBD)")
	}
	return nil
}

func (e *Enforcer) applyEBPFRules(rules RuleSet) error {
	if err := e.ebpfMgr.ClearBlocklist(); err != nil {
		return fmt.Errorf("clear blocklist: %w", err)
	}
	if err := e.ebpfMgr.ClearAllowlist(); err != nil {
		return fmt.Errorf("clear allowlist: %w", err)
	}

	for _, cidr := range rules.BlockCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			logEnforcer.WithField("cidr", cidr).Warn("levers: invalid block CIDR, skipping")
			continue
		}
		if err := e.ebpfMgr.BlockCIDR(cidr); err != nil {
			return fmt.Errorf("block CIDR %s: %w", cidr, err)
		}
	}

	for _, cidr := range rules.AllowCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			logEnforcer.WithField("cidr", cidr).Warn("levers: invalid allow CIDR, skipping")
			continue
		}
		if err := e.ebpfMgr.AllowCIDR(cidr); err != nil {
			return fmt.Errorf("allow CIDR %s: %w", cidr, err)
		}
	}

	for srcIP, pps := range rules.RateLimits {
		if err := e.ebpfMgr.SetRateLimit(srcIP, pps); err != nil {
			logEnforcer.WithField("src_ip", srcIP).Warn("levers: failed to set rate limit, continuing")
		}
	}
	return nil
}
