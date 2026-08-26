# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f api/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY api/go.mod api/go.sum ./api/
RUN cd api && go mod download
COPY api/ ./api/
RUN cd api && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
# Serves HTTP (HTTP_ADDR default :8090); /metrics + /healthz + /readyz too.
EXPOSE 8090
USER nonroot:nonroot
ENTRYPOINT ["/api"]
