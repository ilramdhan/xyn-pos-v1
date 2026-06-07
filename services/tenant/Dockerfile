# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace

# Copy workspace files first for layer caching
COPY go.work go.work.sum ./
COPY shared/go/go.mod shared/go/go.sum ./shared/go/
COPY gen/go.mod gen/go.sum ./gen/
COPY services/tenant/go.mod services/tenant/go.sum ./services/tenant/

# Download dependencies
RUN go work sync

# Copy source code
COPY shared/go/ ./shared/go/
COPY gen/ ./gen/
COPY services/tenant/ ./services/tenant/

# Build — static binary, no CGO
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /service \
    ./services/tenant/cmd/server/

# Runtime — minimal distroless image, non-root
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
