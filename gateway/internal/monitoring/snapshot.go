package monitoring

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Service is a single row in the monitoring snapshot.
type Service struct {
	Name         string  `json:"name"`         // prometheus job, e.g. "auth-service"
	DisplayName  string  `json:"displayName"`  // user-facing label
	Up           bool    `json:"up"`           // up == 1 in last scrape
	RPS          float64 `json:"rps"`          // requests/second over the last minute
	LatencyP95Ms float64 `json:"latencyP95Ms"` // p95 grpc/http handling latency
	ErrorRate    float64 `json:"errorRate"`    // 0.0..1.0
}

// Snapshot is the aggregated payload returned by the monitoring endpoint.
type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`
	Services  []Service `json:"services"`
}

// targetSpec describes how to query metrics for one service. Each target has
// independent PromQL expressions because Auth/TwoFA expose `grpc_*` metrics
// while Gateway exposes `http_*`.
type targetSpec struct {
	Name        string
	DisplayName string
	UpQuery     string
	RPSQuery    string
	P95Query    string
	ErrQuery    string
}

// builtinTargets is the fixed allowlist of services the snapshot covers.
// Adding a new service means appending here — never accepting a query name
// from the request.
//
// Metric prefixes follow the project's naming convention exposed via
// internal/middleware/metrics.go in each service:
//
//   - auth_requests_total / auth_request_duration_seconds_*
//   - twofa_request_duration_seconds_* (no separate counter — _count works)
//   - mpc_request_duration_seconds_* (per node — disambiguated by job label)
//   - gateway_http_requests_total / gateway_http_request_duration_seconds_*
var builtinTargets = []targetSpec{
	grpcTarget("auth-service", "Auth Service", "auth", true),
	grpcTarget("twofa-service", "TwoFA Service", "twofa", false),
	grpcTarget("mpc-node-1", "MPC Node 1", "mpc", false),
	grpcTarget("mpc-node-2", "MPC Node 2", "mpc", false),
	grpcTarget("mpc-node-3", "MPC Node 3", "mpc", false),
	{
		Name: "gateway", DisplayName: "API Gateway",
		UpQuery:  `up{job="gateway"}`,
		RPSQuery: `sum(rate(gateway_http_requests_total{job="gateway"}[1m]))`,
		P95Query: `histogram_quantile(0.95, sum(rate(gateway_http_request_duration_seconds_bucket{job="gateway"}[1m])) by (le))`,
		ErrQuery: `sum(rate(gateway_http_requests_total{job="gateway",status=~"5.."}[1m])) / clamp_min(sum(rate(gateway_http_requests_total{job="gateway"}[1m])), 1)`,
	},
}

// grpcTarget builds queries for an in-house gRPC service (auth/twofa/mpc).
// hasStatusLabel is true when the service exposes a counter with a `status`
// label (only auth in the current codebase) — that's the only one where we
// can compute a real error rate; the rest report 0.
func grpcTarget(job, display, prefix string, hasStatusLabel bool) targetSpec {
	t := targetSpec{
		Name: job, DisplayName: display,
		UpQuery: fmt.Sprintf(`up{job=%q}`, job),
		RPSQuery: fmt.Sprintf(
			`sum(rate(%s_request_duration_seconds_count{job=%q}[1m]))`, prefix, job,
		),
		P95Query: fmt.Sprintf(
			`histogram_quantile(0.95, sum(rate(%s_request_duration_seconds_bucket{job=%q}[1m])) by (le))`,
			prefix, job,
		),
	}
	if hasStatusLabel {
		t.ErrQuery = fmt.Sprintf(
			`sum(rate(%s_requests_total{job=%q,status!="OK"}[1m])) / clamp_min(sum(rate(%s_requests_total{job=%q}[1m])), 1)`,
			prefix, job, prefix, job,
		)
	}
	return t
}

// Collector orchestrates Prometheus queries for the snapshot endpoint.
type Collector struct {
	prom *PromClient
}

// NewCollector returns a Collector wired to a PromClient.
func NewCollector(prom *PromClient) *Collector {
	return &Collector{prom: prom}
}

// Build queries Prometheus for every target in parallel and returns an
// aggregated snapshot. Targets that error out individually still appear in
// the response with Up=false and zero metrics — partial results beat
// failing the whole endpoint.
func (c *Collector) Build(ctx context.Context) (Snapshot, error) {
	if c == nil || c.prom == nil {
		return Snapshot{}, errors.New("monitoring collector not configured")
	}

	out := make([]Service, len(builtinTargets))
	g, gCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	for i, t := range builtinTargets {
		g.Go(func() error {
			svc := c.querySingle(gCtx, t)
			mu.Lock()
			out[i] = svc
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait() // individual failures are absorbed inside querySingle.

	return Snapshot{Timestamp: time.Now().UTC(), Services: out}, nil
}

func (c *Collector) querySingle(ctx context.Context, t targetSpec) Service {
	svc := Service{Name: t.Name, DisplayName: t.DisplayName}

	if v, ok, _ := c.prom.Query(ctx, t.UpQuery); ok {
		svc.Up = v >= 1
	}
	if v, ok, _ := c.prom.Query(ctx, t.RPSQuery); ok {
		svc.RPS = v
	}
	if v, ok, _ := c.prom.Query(ctx, t.P95Query); ok {
		svc.LatencyP95Ms = v * 1000 // Prometheus returns seconds.
	}
	if v, ok, _ := c.prom.Query(ctx, t.ErrQuery); ok {
		svc.ErrorRate = v
	}
	return svc
}
