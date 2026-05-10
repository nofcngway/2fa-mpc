package bootstrap

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// loadServerTLSCredentials returns gRPC server transport credentials configured
// for mTLS: TwoFA presents its own cert and requires every Gateway client to
// present a cert signed by the CA in caFile.
func loadServerTLSCredentials(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, errors.New("tls cert_file, key_file and ca_file are required when tls.enabled=true")
	}

	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server keypair: %w", err)
	}

	caPool, err := loadCAPool(caFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}), nil
}

// loadClientTLSCredentials returns gRPC client transport credentials configured
// for mTLS: TwoFA presents its own cert when dialing an MPC node and validates
// the server cert against the CA in caFile. The dial target is used as the
// expected server name (must match a SAN in the server cert).
func loadClientTLSCredentials(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, errors.New("tls cert_file, key_file and ca_file are required when tls.enabled=true")
	}

	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client keypair: %w", err)
	}

	caPool, err := loadCAPool(caFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}), nil
}

func loadCAPool(caFile string) (*x509.CertPool, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read ca file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to append CA cert to pool")
	}
	return pool, nil
}
