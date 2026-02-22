package netutil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

// LoadMTLSConfig loads a client/server certificate and a CA certificate to configure
// mutual TLS (mTLS) for secure node-to-node communication over TCP/gRPC.
func LoadMTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load the node's certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	// Load CA public key to verify incoming and outgoing connections
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to append CA cert to pool")
	}

	// Configure mTLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool, // Client mode: uses this to verify the server
		ClientCAs:    caCertPool, // Server mode: uses this to verify the client
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	return tlsConfig, nil
}
