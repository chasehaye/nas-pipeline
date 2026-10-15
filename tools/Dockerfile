# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f tools/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY tools/go.mod tools/go.sum ./tools/
RUN cd tools && go mod download
COPY tools/ ./tools/
RUN cd tools && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/census ./cmd/census

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/census /census
USER nonroot:nonroot
ENTRYPOINT ["/census"]
