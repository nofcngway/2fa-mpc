package bootstrap

import (
	"fmt"
	"io"

	"google.golang.org/grpc"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/adapters/mpcclient"
	"github.com/vbncursed/vkr/twofa/internal/middleware"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

// NewMPCClients dials all MPC nodes from config and wraps each connection as
// a domain MPCClient. Returns the client slice and a parallel slice of
// io.Closers for orderly shutdown. On any dial failure the caller is given
// nothing — already-opened connections are closed before returning.
func NewMPCClients(cfg *config.Config) ([]twofaService.MPCClient, []io.Closer, error) {
	transportCreds, err := mpcTransportCreds(cfg)
	if err != nil {
		return nil, nil, err
	}

	clients := make([]twofaService.MPCClient, len(cfg.MPCNodes))
	conns := make([]io.Closer, len(cfg.MPCNodes))

	for i, node := range cfg.MPCNodes {
		conn, err := grpc.NewClient(node.Addr,
			transportCreds,
			grpc.WithUnaryInterceptor(middleware.ClientAuthInterceptor(cfg.SharedSecret)),
		)
		if err != nil {
			// Best-effort cleanup of already-opened connections; close errors
			// during failed dial are not actionable for the caller.
			for j := range i {
				_ = conns[j].Close()
			}
			return nil, nil, fmt.Errorf("connect to MPC node %d at %s: %w", i, node.Addr, err)
		}
		clients[i] = mpcclient.New(conn)
		conns[i] = conn
	}
	return clients, conns, nil
}
