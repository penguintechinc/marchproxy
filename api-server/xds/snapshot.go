// Package xds provides snapshot generation for Envoy xDS configuration
package main

import (
	"encoding/json"
	"fmt"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	http_conn "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ServiceConfig represents a service configuration from the API server
type ServiceConfig struct {
	Name            string   `json:"name"`
	Hosts           []string `json:"hosts"`
	Port            uint32   `json:"port"`
	Protocol        string   `json:"protocol"` // http, https, grpc, http2, websocket
	TLSEnabled      bool     `json:"tls_enabled"`
	TLSCertName     string   `json:"tls_cert_name"`     // Reference to certificate
	TLSVerify       bool     `json:"tls_verify"`        // Verify upstream TLS
	HealthCheckPath string   `json:"health_check_path"` // Health check endpoint
	TimeoutSeconds  int      `json:"timeout_seconds"`   // Connection timeout
	HTTP2Enabled    bool     `json:"http2_enabled"`     // Enable HTTP/2
	WebSocketUpgrade bool    `json:"websocket_upgrade"` // Enable WebSocket upgrade
}

// RouteConfig represents a route configuration
type RouteConfig struct {
	Name         string   `json:"name"`
	Prefix       string   `json:"prefix"`
	ClusterName  string   `json:"cluster_name"`
	Hosts        []string `json:"hosts"`
	Timeout      int      `json:"timeout"` // seconds
}

// CertificateConfig represents TLS certificate configuration
type CertificateConfig struct {
	Name         string `json:"name"`
	CertChain    string `json:"cert_chain"`    // PEM encoded certificate chain
	PrivateKey   string `json:"private_key"`   // PEM encoded private key
	CACert       string `json:"ca_cert"`       // Optional CA certificate
	RequireClient bool  `json:"require_client"` // Require client certificates
}

// MarchProxyConfig represents the complete configuration from API server
type MarchProxyConfig struct {
	Version      string              `json:"version"`
	Services     []ServiceConfig     `json:"services"`
	Routes       []RouteConfig       `json:"routes"`
	Certificates []CertificateConfig `json:"certificates"`
}

// GenerateSnapshot creates an Envoy snapshot from MarchProxy configuration
func GenerateSnapshot(config MarchProxyConfig) (*cache.Snapshot, error) {
	// Create clusters with enhanced configuration
	var clusters []types.Resource
	for _, svc := range config.Services {
		c, err := makeClusterWithConfig(svc)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster for %s: %w", svc.Name, err)
		}
		clusters = append(clusters, c)
	}

	// Create endpoints
	var endpoints []types.Resource
	for _, svc := range config.Services {
		e := makeEndpoint(svc)
		endpoints = append(endpoints, e)
	}

	// Create routes
	var routes []types.Resource
	if len(config.Routes) > 0 {
		r, err := makeRouteConfiguration(config.Routes)
		if err != nil {
			return nil, fmt.Errorf("failed to create routes: %w", err)
		}
		routes = append(routes, r)
	}

	// Create listeners
	var listeners []types.Resource
	l, err := makeListener(config.Routes, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	listeners = append(listeners, l)

	// Create secrets for TLS certificates (SDS)
	var secrets []types.Resource
	if len(config.Certificates) > 0 {
		s, err := makeSecrets(config.Certificates)
		if err != nil {
			return nil, fmt.Errorf("failed to create secrets: %w", err)
		}
		secrets = s
	}

	// Create snapshot with all resources
	resourceMap := map[resource.Type][]types.Resource{
		resource.EndpointType: endpoints,
		resource.ClusterType:  clusters,
		resource.RouteType:    routes,
		resource.ListenerType: listeners,
	}

	// Only add secrets if we have them
	if len(secrets) > 0 {
		resourceMap[resource.SecretType] = secrets
	}

	snapshot, err := cache.NewSnapshot(config.Version, resourceMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	return snapshot, nil
}

// makeCluster creates an Envoy cluster from a service configuration
func makeCluster(svc ServiceConfig) (*cluster.Cluster, error) {
	return &cluster.Cluster{
		Name:                 svc.Name,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:       makeEndpoint(svc),
		DnsLookupFamily:      cluster.Cluster_V4_ONLY,
	}, nil
}

// makeEndpoint creates an Envoy endpoint from a service configuration
func makeEndpoint(svc ServiceConfig) *endpoint.ClusterLoadAssignment {
	var lbEndpoints []*endpoint.LbEndpoint

	for _, host := range svc.Hosts {
		lbEndpoints = append(lbEndpoints, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  host,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: svc.Port,
								},
							},
						},
					},
				},
			},
		})
	}

	return &endpoint.ClusterLoadAssignment{
		ClusterName: svc.Name,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: lbEndpoints,
		}},
	}
}

// makeRouteConfiguration creates an Envoy route configuration
func makeRouteConfiguration(routes []RouteConfig) (*route.RouteConfiguration, error) {
	var virtualHosts []*route.VirtualHost

	// Group routes by host
	hostRoutes := make(map[string][]*route.Route)
	for _, r := range routes {
		for _, host := range r.Hosts {
			timeout := 30 * time.Second
			if r.Timeout > 0 {
				timeout = time.Duration(r.Timeout) * time.Second
			}

			route := &route.Route{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: r.Prefix,
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: r.ClusterName,
						},
						Timeout: durationpb.New(timeout),
					},
				},
			}
			hostRoutes[host] = append(hostRoutes[host], route)
		}
	}

	// Create virtual hosts
	for host, routes := range hostRoutes {
		virtualHosts = append(virtualHosts, &route.VirtualHost{
			Name:    host,
			Domains: []string{host, host + ":*"},
			Routes:  routes,
		})
	}

	// Add default catch-all virtual host if no routes specified
	if len(virtualHosts) == 0 {
		virtualHosts = append(virtualHosts, &route.VirtualHost{
			Name:    "default",
			Domains: []string{"*"},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_DirectResponse{
					DirectResponse: &route.DirectResponseAction{
						Status: 404,
					},
				},
			}},
		})
	}

	return &route.RouteConfiguration{
		Name:         "marchproxy_routes",
		VirtualHosts: virtualHosts,
	}, nil
}

// makeListener creates an Envoy listener for HTTP traffic with WebSocket and HTTP/2 support
func makeListener(routes []RouteConfig, config MarchProxyConfig) (*listener.Listener, error) {
	// Determine if we need WebSocket or HTTP/2 support
	websocketEnabled := false
	http2Enabled := false

	for _, svc := range config.Services {
		if svc.WebSocketUpgrade {
			websocketEnabled = true
		}
		if svc.HTTP2Enabled || svc.Protocol == "grpc" || svc.Protocol == "http2" {
			http2Enabled = true
		}
	}

	// Create HTTP filters - use first service config for filter generation
	var httpFilters []*http_conn.HttpFilter
	if len(config.Services) > 0 {
		filters, err := makeHTTPFilters(config.Services[0])
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP filters: %w", err)
		}
		httpFilters = filters
	} else {
		// Default router filter
		httpFilters = []*http_conn.HttpFilter{{
			Name: wellknown.Router,
		}}
	}

	// Create HTTP connection manager with WebSocket and HTTP/2 support
	manager := &http_conn.HttpConnectionManager{
		CodecType:  http_conn.HttpConnectionManager_AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &http_conn.HttpConnectionManager_Rds{
			Rds: &http_conn.Rds{
				ConfigSource: &core.ConfigSource{
					ResourceApiVersion: core.ApiVersion_V3,
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: "marchproxy_routes",
			},
		},
		HttpFilters: httpFilters,
	}

	// Add WebSocket upgrade config if needed
	if websocketEnabled {
		manager.UpgradeConfigs = makeUpgradeConfigs(true)
	}

	// Add HTTP/2 options if needed
	if http2Enabled {
		manager.Http2ProtocolOptions = &core.Http2ProtocolOptions{
			AllowConnect:  true,
			AllowMetadata: true,
		}
	}

	pbst, err := anypb.New(manager)
	if err != nil {
		return nil, err
	}

	return &listener.Listener{
		Name: "marchproxy_listener",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 10000,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}, nil
}

// ParseConfig parses JSON configuration into MarchProxyConfig
func ParseConfig(data []byte) (*MarchProxyConfig, error) {
	var config MarchProxyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// makeSecrets creates Envoy secrets for TLS certificates (SDS)
func makeSecrets(certificates []CertificateConfig) ([]types.Resource, error) {
	var secrets []types.Resource

	for _, cert := range certificates {
		// Create TLS certificate secret
		tlsCert := &core.DataSource{
			Specifier: &core.DataSource_InlineString{
				InlineString: cert.CertChain,
			},
		}

		tlsKey := &core.DataSource{
			Specifier: &core.DataSource_InlineString{
				InlineString: cert.PrivateKey,
			},
		}

		secret := &tls.Secret{
			Name: cert.Name,
			Type: &tls.Secret_TlsCertificate{
				TlsCertificate: &tls.TlsCertificate{
					CertificateChain: tlsCert,
					PrivateKey:       tlsKey,
				},
			},
		}

		secrets = append(secrets, secret)

		// If CA cert is provided, create validation context secret
		if cert.CACert != "" {
			caCert := &core.DataSource{
				Specifier: &core.DataSource_InlineString{
					InlineString: cert.CACert,
				},
			}

			validationSecret := &tls.Secret{
				Name: cert.Name + "_validation",
				Type: &tls.Secret_ValidationContext{
					ValidationContext: &tls.CertificateValidationContext{
						TrustedCa: caCert,
					},
				},
			}

			secrets = append(secrets, validationSecret)
		}
	}

	return secrets, nil
}
