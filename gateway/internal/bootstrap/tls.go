package bootstrap

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// loadClientTLSCredentials returns gRPC client transport credentials configured
// for mTLS: the Gateway presents its own cert when dialing Auth/TwoFA and
// validates the server cert against the CA in caFile. The dial target's
// hostname is used as the expected server name (must match a SAN in the
// server cert).
func loadClientTLSCredentials(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, errors.New("tls cert_file, key_file and ca_file are required when tls.enabled=true")
	}

	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client keypair: %w", err)
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read ca file: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to append CA cert to pool")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}), nil
}
