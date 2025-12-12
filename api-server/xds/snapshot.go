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
	Name     string   `json:"name"`
	Hosts    []string `json:"hosts"`
	Port     uint32   `json:"port"`
	Protocol string   `json:"protocol"` // http, https, grpc
}

// RouteConfig represents a route configuration
type RouteConfig struct {
	Name         string   `json:"name"`
	Prefix       string   `json:"prefix"`
	ClusterName  string   `json:"cluster_name"`
	Hosts        []string `json:"hosts"`
	Timeout      int      `json:"timeout"` // seconds
}

// MarchProxyConfig represents the complete configuration from API server
type MarchProxyConfig struct {
	Version  string          `json:"version"`
	Services []ServiceConfig `json:"services"`
	Routes   []RouteConfig   `json:"routes"`
}

// GenerateSnapshot creates an Envoy snapshot from MarchProxy configuration
func GenerateSnapshot(config MarchProxyConfig) (*cache.Snapshot, error) {
	// Create clusters
	var clusters []types.Resource
	for _, svc := range config.Services {
		c, err := makeCluster(svc)
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
	l, err := makeListener(config.Routes)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	listeners = append(listeners, l)

	// Create snapshot
	snapshot, err := cache.NewSnapshot(
		config.Version,
		map[resource.Type][]types.Resource{
			resource.EndpointType: endpoints,
			resource.ClusterType:  clusters,
			resource.RouteType:    routes,
			resource.ListenerType: listeners,
		},
	)
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

// makeListener creates an Envoy listener for HTTP traffic
func makeListener(routes []RouteConfig) (*listener.Listener, error) {
	// Create HTTP connection manager
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
		HttpFilters: []*http_conn.HttpFilter{{
			Name: wellknown.Router,
		}},
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
