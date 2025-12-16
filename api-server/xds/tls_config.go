// Package xds provides TLS configuration for Envoy
package main

import (
	"fmt"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
)

// makeDownstreamTLSContext creates TLS context for listener (downstream)
func makeDownstreamTLSContext(certName string, requireClientCert bool) (*core.TransportSocket, error) {
	tlsContext := &tls.DownstreamTlsContext{
		CommonTlsContext: &tls.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*tls.SdsSecretConfig{
				{
					Name: certName,
					SdsConfig: &core.ConfigSource{
						ResourceApiVersion: core.ApiVersion_V3,
						ConfigSourceSpecifier: &core.ConfigSource_Ads{
							Ads: &core.AggregatedConfigSource{},
						},
					},
				},
			},
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
	}

	// Add client certificate validation if required
	if requireClientCert {
		tlsContext.RequireClientCertificate = wrappers.Bool(true)
		tlsContext.CommonTlsContext.ValidationContextType = &tls.CommonTlsContext_ValidationContext{
			ValidationContext: &tls.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/etc/envoy/ca-cert.pem",
					},
				},
			},
		}
	}

	pbst, err := anypb.New(tlsContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TLS context: %w", err)
	}

	return &core.TransportSocket{
		Name: wellknown.TransportSocketTls,
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: pbst,
		},
	}, nil
}

// makeUpstreamTLSContext creates TLS context for cluster (upstream)
func makeUpstreamTLSContext(certName string, verifyPeer bool, sni string) (*core.TransportSocket, error) {
	tlsContext := &tls.UpstreamTlsContext{
		Sni: sni,
		CommonTlsContext: &tls.CommonTlsContext{
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
	}

	// Add client certificate if provided
	if certName != "" {
		tlsContext.CommonTlsContext.TlsCertificateSdsSecretConfigs = []*tls.SdsSecretConfig{
			{
				Name: certName,
				SdsConfig: &core.ConfigSource{
					ResourceApiVersion: core.ApiVersion_V3,
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
			},
		}
	}

	// Add validation context if peer verification is required
	if verifyPeer {
		tlsContext.CommonTlsContext.ValidationContextType = &tls.CommonTlsContext_ValidationContext{
			ValidationContext: &tls.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/etc/envoy/ca-cert.pem",
					},
				},
			},
		}
	}

	pbst, err := anypb.New(tlsContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal upstream TLS context: %w", err)
	}

	return &core.TransportSocket{
		Name: wellknown.TransportSocketTls,
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: pbst,
		},
	}, nil
}

// makeTLSInspectorFilter creates TLS inspector filter for SNI detection
func makeTLSInspectorFilter() (*core.Address, error) {
	// TLS inspector is configured as a listener filter, not a network filter
	// This is a placeholder for documentation purposes
	return nil, nil
}
