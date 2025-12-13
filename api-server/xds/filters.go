// Package xds provides HTTP filter configurations for Envoy
package main

import (
	"fmt"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	cors "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	grpc_web "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	http_conn "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"time"
)

// makeHTTPFilters creates HTTP filter chain with WebSocket and HTTP/2 support
func makeHTTPFilters(config ServiceConfig) ([]*http_conn.HttpFilter, error) {
	filters := []*http_conn.HttpFilter{}

	// Add CORS filter for HTTP services
	if config.Protocol == "http" || config.Protocol == "https" || config.Protocol == "http2" {
		corsFilter, err := makeCORSFilter()
		if err != nil {
			return nil, fmt.Errorf("failed to create CORS filter: %w", err)
		}
		filters = append(filters, corsFilter)
	}

	// Add gRPC-Web filter for gRPC services
	if config.Protocol == "grpc" {
		grpcWebFilter, err := makeGRPCWebFilter()
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC-Web filter: %w", err)
		}
		filters = append(filters, grpcWebFilter)
	}

	// Add router filter (always last)
	routerFilter, err := makeRouterFilter()
	if err != nil {
		return nil, fmt.Errorf("failed to create router filter: %w", err)
	}
	filters = append(filters, routerFilter)

	return filters, nil
}

// makeCORSFilter creates CORS filter configuration
func makeCORSFilter() (*http_conn.HttpFilter, error) {
	corsConfig := &cors.Cors{}

	pbst, err := anypb.New(corsConfig)
	if err != nil {
		return nil, err
	}

	return &http_conn.HttpFilter{
		Name: wellknown.CORS,
		ConfigType: &http_conn.HttpFilter_TypedConfig{
			TypedConfig: pbst,
		},
	}, nil
}

// makeGRPCWebFilter creates gRPC-Web filter for gRPC services
func makeGRPCWebFilter() (*http_conn.HttpFilter, error) {
	grpcWebConfig := &grpc_web.GrpcWeb{}

	pbst, err := anypb.New(grpcWebConfig)
	if err != nil {
		return nil, err
	}

	return &http_conn.HttpFilter{
		Name: wellknown.GRPCWeb,
		ConfigType: &http_conn.HttpFilter_TypedConfig{
			TypedConfig: pbst,
		},
	}, nil
}

// makeRouterFilter creates the router filter (always last in chain)
func makeRouterFilter() (*http_conn.HttpFilter, error) {
	routerConfig := &router.Router{}

	pbst, err := anypb.New(routerConfig)
	if err != nil {
		return nil, err
	}

	return &http_conn.HttpFilter{
		Name: wellknown.Router,
		ConfigType: &http_conn.HttpFilter_TypedConfig{
			TypedConfig: pbst,
		},
	}, nil
}

// makeUpgradeConfigs creates upgrade configurations for WebSocket
func makeUpgradeConfigs(websocketEnabled bool) []*http_conn.HttpConnectionManager_UpgradeConfig {
	if !websocketEnabled {
		return nil
	}

	return []*http_conn.HttpConnectionManager_UpgradeConfig{
		{
			UpgradeType: "websocket",
			Enabled:     wrapperspb.Bool(true),
		},
	}
}

// makeHTTP2Options creates HTTP/2 protocol options
func makeHTTP2Options(http2Enabled bool) *core.Http2ProtocolOptions {
	if !http2Enabled {
		return nil
	}

	return &core.Http2ProtocolOptions{
		// Enable HTTP/2 with default settings
		AllowConnect:  true,
		AllowMetadata: true,
	}
}

// makeRouteMatchWithWebSocket creates route match configuration with WebSocket support
func makeRouteMatchWithWebSocket(prefix string, websocketEnabled bool) *route.RouteMatch {
	match := &route.RouteMatch{
		PathSpecifier: &route.RouteMatch_Prefix{
			Prefix: prefix,
		},
	}

	// Add WebSocket upgrade matcher if enabled
	if websocketEnabled {
		match.Headers = []*route.HeaderMatcher{
			{
				Name: "upgrade",
				HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
					ExactMatch: "websocket",
				},
			},
		}
	}

	return match
}

// makeHealthCheckConfig creates health check configuration for clusters
func makeHealthCheckConfig(healthCheckPath string, protocol string) *core.HealthCheck {
	if healthCheckPath == "" {
		return nil
	}

	// Create HTTP health check
	httpHealthCheck := &core.HealthCheck_HttpHealthCheck{
		Path: healthCheckPath,
	}

	// Add gRPC headers if needed
	if protocol == "grpc" {
		httpHealthCheck.RequestHeadersToAdd = []*core.HeaderValueOption{
			{
				Header: &core.HeaderValue{
					Key:   "content-type",
					Value: "application/grpc",
				},
			},
		}
	}

	return &core.HealthCheck{
		Timeout:            durationpb.New(5 * time.Second),
		Interval:           durationpb.New(10 * time.Second),
		UnhealthyThreshold: wrapperspb.UInt32(3),
		HealthyThreshold:   wrapperspb.UInt32(2),
		HealthChecker: &core.HealthCheck_HttpHealthCheck_{
			HttpHealthCheck: httpHealthCheck,
		},
	}
}

// makeClusterWithConfig creates an enhanced cluster with health checks and TLS
func makeClusterWithConfig(svc ServiceConfig) (*cluster.Cluster, error) {
	// Base cluster configuration
	c := &cluster.Cluster{
		Name:                 svc.Name,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		DnsLookupFamily:      cluster.Cluster_V4_ONLY,
	}

	// Override timeout if specified
	if svc.TimeoutSeconds > 0 {
		c.ConnectTimeout = durationpb.New(time.Duration(svc.TimeoutSeconds) * time.Second)
	}

	// Add health check if configured
	if svc.HealthCheckPath != "" {
		healthCheck := makeHealthCheckConfig(svc.HealthCheckPath, svc.Protocol)
		if healthCheck != nil {
			c.HealthChecks = []*core.HealthCheck{healthCheck}
		}
	}

	// Add HTTP/2 protocol options if enabled
	if svc.HTTP2Enabled || svc.Protocol == "grpc" || svc.Protocol == "http2" {
		c.Http2ProtocolOptions = &core.Http2ProtocolOptions{}
	}

	// Add TLS transport socket if TLS is enabled
	if svc.TLSEnabled && svc.TLSCertName != "" {
		// Determine SNI from first host
		sni := ""
		if len(svc.Hosts) > 0 {
			sni = svc.Hosts[0]
		}

		tlsSocket, err := makeUpstreamTLSContext(svc.TLSCertName, svc.TLSVerify, sni)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS context: %w", err)
		}
		c.TransportSocket = tlsSocket
	}

	return c, nil
}
