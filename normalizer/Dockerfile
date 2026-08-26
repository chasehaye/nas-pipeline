# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f normalizer/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY normalizer/go.mod normalizer/go.sum ./normalizer/
RUN cd normalizer && go mod download
COPY normalizer/ ./normalizer/
RUN cd normalizer && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/normalizer ./cmd/normalizer

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/normalizer /normalizer
USER nonroot:nonroot
ENTRYPOINT ["/normalizer"]
