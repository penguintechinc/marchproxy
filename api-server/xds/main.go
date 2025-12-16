package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

const (
	grpcKeepaliveTime        = 30 * time.Second
	grpcKeepaliveTimeout     = 5 * time.Second
	grpcKeepaliveMinTime     = 30 * time.Second
	grpcMaxConcurrentStreams = 1000000
)

var (
	port        = flag.Int("port", 18000, "xDS management server port")
	nodeID      = flag.String("nodeID", "marchproxy-control-plane", "Node ID")
	debug       = flag.Bool("debug", false, "Enable debug logging")
	metricsPort = flag.Int("metrics", 19000, "Metrics server port")
)

func main() {
	flag.Parse()

	// Create snapshot cache
	cache := NewSnapshotCache(*debug)

	// Create xDS server callbacks
	cb := &Callbacks{
		Signal:   make(chan struct{}),
		Fetches:  0,
		Requests: 0,
		Debug:    *debug,
	}

	// Create the xDS server
	srv := server.NewServer(context.Background(), cache, cb)

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)

	// Register xDS services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, srv)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, srv)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, srv)

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("xDS management server listening on :%d\n", *port)

	// Start metrics server
	go startMetricsServer(*metricsPort, cache, cb)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down xDS server...")
		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serve: %v\n", err)
		os.Exit(1)
	}
}

// startMetricsServer starts HTTP server for health checks, metrics, and config API
func startMetricsServer(port int, cache *SnapshotCache, cb *Callbacks) {
	// Create config API
	configAPI := NewConfigAPI(cache, *nodeID)

	// Setup HTTP server
	mux := http.NewServeMux()

	// Configuration management endpoints
	mux.HandleFunc("/v1/config", configAPI.UpdateConfigHandler)
	mux.HandleFunc("/v1/version", configAPI.GetConfigHandler)
	mux.HandleFunc("/v1/snapshot/", configAPI.GetSnapshotHandler)
	mux.HandleFunc("/v1/rollback/", configAPI.RollbackHandler)

	// Health and metrics endpoints
	mux.HandleFunc("/health", configAPI.HealthHandler)
	mux.HandleFunc("/healthz", configAPI.HealthHandler)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# HELP xds_requests_total Total number of xDS requests\n")
		fmt.Fprintf(w, "# TYPE xds_requests_total counter\n")
		fmt.Fprintf(w, "xds_requests_total %d\n", cb.GetRequestCount())
		fmt.Fprintf(w, "# HELP xds_fetches_total Total number of xDS fetches\n")
		fmt.Fprintf(w, "# TYPE xds_fetches_total counter\n")
		fmt.Fprintf(w, "xds_fetches_total %d\n", cb.GetFetchCount())
		fmt.Fprintf(w, "# HELP xds_cache_version Current cache version\n")
		fmt.Fprintf(w, "# TYPE xds_cache_version gauge\n")
		fmt.Fprintf(w, "xds_cache_version %d\n", cache.GetVersion())
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("HTTP API and metrics server listening on %s\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start metrics server: %v\n", err)
	}
}
