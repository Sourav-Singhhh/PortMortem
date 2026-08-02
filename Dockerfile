# Port Mortem 2026 — Standalone Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

# Build port package and adapter binary
WORKDIR /app/port
RUN go build -v ./...
RUN go test -c -o port.test ./...

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/port/port.test /app/port.test

CMD ["/app/port.test", "-test.v"]
