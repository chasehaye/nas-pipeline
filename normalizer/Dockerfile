# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26.1 AS build
WORKDIR /src
# Copy manifests first so `go mod download` is cached across code-only changes.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static binary (CGO off) so it runs in a distroless/scratch image.
RUN CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/normalizer ./cmd/normalizer

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/normalizer /normalizer
USER nonroot:nonroot
ENTRYPOINT ["/normalizer"]
