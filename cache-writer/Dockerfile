# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f cache-writer/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY cache-writer/go.mod cache-writer/go.sum ./cache-writer/
RUN cd cache-writer && go mod download
COPY cache-writer/ ./cache-writer/
RUN cd cache-writer && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/cache-writer ./cmd/cache-writer

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/cache-writer /cache-writer
USER nonroot:nonroot
ENTRYPOINT ["/cache-writer"]
