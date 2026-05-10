package bootstrap

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gotest.tools/v3/assert"
)

// repoCertsPath returns the absolute path to the project's dev PKI directory.
// Tests skip when scripts/gen-certs.sh has not been run.
func repoCertsPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	assert.NilError(t, err)
	// twofa/internal/bootstrap → repo root is three levels up.
	root := filepath.Join(wd, "..", "..", "..")
	certsDir := filepath.Join(root, "certs")
	if _, err := os.Stat(filepath.Join(certsDir, "ca.crt")); os.IsNotExist(err) {
		t.Skip("dev PKI missing — run scripts/gen-certs.sh")
	}
	return certsDir
}

func TestLoadServerTLSCredentials(t *testing.T) {
	certs := repoCertsPath(t)
	creds, err := loadServerTLSCredentials(
		filepath.Join(certs, "twofa.crt"),
		filepath.Join(certs, "twofa.key"),
		filepath.Join(certs, "ca.crt"),
	)
	assert.NilError(t, err)
	assert.Assert(t, creds != nil)
}

func TestLoadClientTLSCredentials(t *testing.T) {
	certs := repoCertsPath(t)
	creds, err := loadClientTLSCredentials(
		filepath.Join(certs, "twofa.crt"),
		filepath.Join(certs, "twofa.key"),
		filepath.Join(certs, "ca.crt"),
	)
	assert.NilError(t, err)
	assert.Assert(t, creds != nil)
}

func TestLoadServerTLSCredentials_RejectsMissingFiles(t *testing.T) {
	_, err := loadServerTLSCredentials("", "", "")
	assert.Assert(t, err != nil, "empty paths must be rejected")

	_, err = loadServerTLSCredentials("/nope/cert.crt", "/nope/key.key", "/nope/ca.crt")
	assert.Assert(t, err != nil, "non-existent files must be rejected")
}

// TestMTLS_EndToEnd asserts a full mTLS handshake against a real gRPC server:
// the server presents twofa.crt + requires a client cert signed by ca.crt;
// the client presents gateway.crt and validates twofa.crt against ca.crt. The
// gRPC health-check RPC succeeds when both sides authenticate correctly.
func TestMTLS_EndToEnd(t *testing.T) {
	certs := repoCertsPath(t)

	serverCreds, err := loadServerTLSCredentials(
		filepath.Join(certs, "twofa.crt"),
		filepath.Join(certs, "twofa.key"),
		filepath.Join(certs, "ca.crt"),
	)
	assert.NilError(t, err)

	clientCreds, err := loadClientTLSCredentials(
		filepath.Join(certs, "gateway.crt"),
		filepath.Join(certs, "gateway.key"),
		filepath.Join(certs, "ca.crt"),
	)
	assert.NilError(t, err)

	// Listen on a random loopback port — keep tests hermetic.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	defer lis.Close()

	srv := grpc.NewServer(grpc.Creds(serverCreds))
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, hs)
	go srv.Serve(lis)
	defer srv.Stop()

	// Loopback dial — the cert SAN must include "localhost".
	host, port, err := net.SplitHostPort(lis.Addr().String())
	assert.NilError(t, err)
	conn, err := grpc.NewClient(net.JoinHostPort("localhost", port),
		grpc.WithTransportCredentials(clientCreds),
		grpc.WithAuthority("twofa"), // override SAN check to match cert
	)
	assert.NilError(t, err)
	defer conn.Close()
	_ = host

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	assert.NilError(t, err, "mTLS handshake must succeed for a properly signed client")
	assert.Equal(t, resp.Status, healthpb.HealthCheckResponse_SERVING)
}
