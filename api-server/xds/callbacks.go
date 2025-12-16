package main

import (
	"context"
	"fmt"
	"sync/atomic"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

// Callbacks implements the server.Callbacks interface
type Callbacks struct {
	Signal   chan struct{}
	Fetches  uint32
	Requests uint32
	Debug    bool
}

var _ server.Callbacks = &Callbacks{}

// OnStreamOpen is called when a new stream is opened
func (cb *Callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	if cb.Debug {
		fmt.Printf("Stream opened: id=%d type=%s\n", id, typ)
	}
	return nil
}

// OnStreamClosed is called when a stream is closed
func (cb *Callbacks) OnStreamClosed(id int64, node *core.Node) {
	if cb.Debug {
		fmt.Printf("Stream closed: id=%d node=%s\n", id, node.GetId())
	}
}

// OnStreamRequest is called when a request is received on a stream
func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
	atomic.AddUint32(&cb.Requests, 1)
	if cb.Debug {
		fmt.Printf("Stream request: id=%d node=%s type=%s version=%s\n",
			id, req.GetNode().GetId(), req.GetTypeUrl(), req.GetVersionInfo())
	}
	return nil
}

// OnStreamResponse is called when a response is sent on a stream
func (cb *Callbacks) OnStreamResponse(ctx context.Context, id int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if cb.Debug {
		fmt.Printf("Stream response: id=%d node=%s type=%s version=%s\n",
			id, req.GetNode().GetId(), req.GetTypeUrl(), resp.GetVersionInfo())
	}
}

// OnFetchRequest is called when a fetch request is received
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *discovery.DiscoveryRequest) error {
	atomic.AddUint32(&cb.Fetches, 1)
	if cb.Debug {
		fmt.Printf("Fetch request: node=%s type=%s\n",
			req.GetNode().GetId(), req.GetTypeUrl())
	}
	return nil
}

// OnFetchResponse is called when a fetch response is sent
func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if cb.Debug {
		fmt.Printf("Fetch response: node=%s type=%s version=%s\n",
			req.GetNode().GetId(), req.GetTypeUrl(), resp.GetVersionInfo())
	}
}

// OnDeltaStreamOpen is called when a delta stream is opened
func (cb *Callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	if cb.Debug {
		fmt.Printf("Delta stream opened: id=%d type=%s\n", id, typ)
	}
	return nil
}

// OnDeltaStreamClosed is called when a delta stream is closed
func (cb *Callbacks) OnDeltaStreamClosed(id int64, node *core.Node) {
	if cb.Debug {
		fmt.Printf("Delta stream closed: id=%d node=%s\n", id, node.GetId())
	}
}

// OnStreamDeltaRequest is called when a delta request is received
func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	if cb.Debug {
		fmt.Printf("Delta request: id=%d node=%s type=%s\n",
			id, req.GetNode().GetId(), req.GetTypeUrl())
	}
	return nil
}

// OnStreamDeltaResponse is called when a delta response is sent
func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, resp *discovery.DeltaDiscoveryResponse) {
	if cb.Debug {
		fmt.Printf("Delta response: id=%d node=%s type=%s\n",
			id, req.GetNode().GetId(), req.GetTypeUrl())
	}
}

// GetRequestCount returns the total number of requests processed
func (cb *Callbacks) GetRequestCount() uint32 {
	return atomic.LoadUint32(&cb.Requests)
}

// GetFetchCount returns the total number of fetch requests processed
func (cb *Callbacks) GetFetchCount() uint32 {
	return atomic.LoadUint32(&cb.Fetches)
}
