# Stage 1: Build the Go binary
FROM registry.access.redhat.com/ubi8/go-toolset:1.21.0 AS builder

WORKDIR /opt/app-root/src
COPY . .

# Build the statically linked binary
RUN CGO_ENABLED=0 go build -o helmify-api ./cmd/helmify-api

# Stage 2: Final runtime image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL org.opencontainers.image.source="https://github.com/danilonicioka/helmify"
LABEL org.opencontainers.image.description="Helmify API - Kubernetes manifest to Helm chart converter"

WORKDIR /app
COPY --from=builder /opt/app-root/src/helmify-api /usr/local/bin/helmify-api

# Ensure the binary is executable
RUN chmod +x /usr/local/bin/helmify-api

# OpenShift requirement: run as non-root
USER 1001

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/helmify-api"]
