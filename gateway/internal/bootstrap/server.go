package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vbncursed/vkr/gateway/config"
	"github.com/vbncursed/vkr/gateway/internal/middleware"
	authpb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
	twofapb "github.com/vbncursed/vkr/gateway/internal/pb/twofa_api"
)

func NewHTTPServer(ctx context.Context, cfg *config.Config, clients *GRPCClients, rdb *redis.Client) (*http.Server, error) {
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, gwMux, cfg.AuthService.Addr, opts); err != nil {
		return nil, fmt.Errorf("register auth gateway: %w", err)
	}
	if err := twofapb.RegisterTwoFAServiceHandlerFromEndpoint(ctx, gwMux, cfg.TwoFAService.Addr, opts); err != nil {
		return nil, fmt.Errorf("register twofa gateway: %w", err)
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /docs", DocsHandler())
	router.HandleFunc("GET /openapi/auth.json", SwaggerFileHandler(cfg.Swagger.Auth))
	router.HandleFunc("GET /openapi/twofa.json", SwaggerFileHandler(cfg.Swagger.TwoFA))
	router.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	router.Handle("/", gwMux)

	// Middleware chain: Recovery → Metrics → Logging → CORS → RateLimit → Auth → Router
	var handler http.Handler = router
	handler = middleware.Auth(clients.Auth)(handler)
	handler = middleware.RateLimit(rdb, cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)(handler)
	handler = middleware.CORS(cfg.CORS.AllowedOrigins)(handler)
	handler = middleware.Logging(handler)
	handler = middleware.Metrics(handler)
	handler = middleware.Recovery(handler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.GetReadTimeout(),
		WriteTimeout: cfg.Server.GetWriteTimeout(),
	}

	slog.Info("HTTP server configured",
		"port", cfg.Server.Port,
		"auth_service", cfg.AuthService.Addr,
		"twofa_service", cfg.TwoFAService.Addr,
	)

	return srv, nil
}
