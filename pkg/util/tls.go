package util

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc/credentials"
)

// LoadServerTLSCredentials loads the server certificate and key, and returns
// gRPC transport credentials for a TLS-enabled server.
func LoadServerTLSCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server TLS credentials: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}
