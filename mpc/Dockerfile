FROM golang:1.26.3 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/app

FROM scratch
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
