### Multi-stage Dockerfile for Hermyx
### Build stage: use official Go image to compile a static binary
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
WORKDIR /src

# Install git for go modules and ca-certificates for HTTPS
RUN apk add --no-cache git build-base ca-certificates

# Copy modules manifests first for efficient layer caching
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Copy the source
COPY . .

# Build a statically linked binary (musl via alpine)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /hermyx ./cmd

### Run stage: minimal image
FROM alpine:3.19 AS runtime
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -S hermyx && adduser -S -G hermyx hermyx

WORKDIR /app
COPY --from=builder /hermyx /app/hermyx
RUN chown hermyx:hermyx /app/hermyx

USER hermyx

# Default config path and storage mount path can be overridden
ENV HERMYX_CONFIG=/configs/hermyx.config.yaml
ENV HERMYX_STORAGE=/data

EXPOSE 8080

VOLUME ["/data", "/configs"]

ENTRYPOINT ["/app/hermyx"]
CMD ["up", "--config", "/configs/hermyx.config.yaml"]
