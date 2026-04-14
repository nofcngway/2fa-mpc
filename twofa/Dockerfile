FROM golang:1.26 AS builder
WORKDIR /build

# Copy both modules (twofa depends on mpc via replace directive)
COPY mpc/go.mod mpc/go.sum ./mpc/
COPY twofa/go.mod twofa/go.sum ./twofa/
WORKDIR /build/twofa
RUN go mod download

WORKDIR /build
COPY mpc/ ./mpc/
COPY twofa/ ./twofa/

WORKDIR /build/twofa
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/app

FROM scratch
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
