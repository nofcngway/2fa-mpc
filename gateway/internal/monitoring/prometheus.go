// Package monitoring queries Prometheus for service-level health snapshots
// surfaced on the dashboard's monitoring page. Only a fixed allowlist of
// PromQL expressions is executed — the Gateway never forwards arbitrary
// queries from the browser.
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PromClient is a tiny client around the Prometheus instant-query HTTP API.
// We avoid the official Prometheus Go SDK to keep transitive deps small.
type PromClient struct {
	baseURL string
	http    *http.Client
}

// NewPromClient builds a client targeting baseURL (e.g. http://prometheus:9090).
// timeout bounds individual queries — set <500ms to keep the snapshot endpoint
// responsive even if Prometheus is overloaded.
func NewPromClient(baseURL string, timeout time.Duration) *PromClient {
	return &PromClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// instantQueryResponse mirrors the relevant subset of the Prometheus
// /api/v1/query response. Only scalar/vector results are decoded; matrices
// are not used by the snapshot endpoint.
type instantQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value [2]any `json:"value"` // [timestamp, "value"]
		} `json:"result"`
	} `json:"data"`
}

// Query runs a single PromQL expression and returns the first vector value as
// a float. Returns (0, false, nil) when Prometheus has no data for the query
// (e.g. a service that has never reported metrics yet).
func (c *PromClient) Query(ctx context.Context, query string) (float64, bool, error) {
	q := url.Values{"query": {query}}
	u := c.baseURL + "/api/v1/query?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, false, fmt.Errorf("build prom request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("prom http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("prom returned %d", resp.StatusCode)
	}

	var body instantQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, false, fmt.Errorf("decode prom: %w", err)
	}
	if body.Status != "success" || len(body.Data.Result) == 0 {
		return 0, false, nil
	}

	// Result[0].Value is [<timestamp:float64>, <value:string>] — Prometheus
	// always serializes the numeric value as a JSON string for precision.
	raw, ok := body.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, false, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false, nil
	}
	// histogram_quantile returns NaN when the histogram has no samples in the
	// rate() window — treat as "no value" so the snapshot reports 0 cleanly
	// instead of failing JSON encoding.
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false, nil
	}
	return v, true, nil
}
