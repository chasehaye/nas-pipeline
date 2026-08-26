# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f filter/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY filter/go.mod filter/go.sum ./filter/
RUN cd filter && go mod download
COPY filter/ ./filter/
RUN cd filter && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/filter ./cmd/filter

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/filter /filter
# LADD Industry file is CUI: it is NOT baked into the image. Mount it at this
# path at runtime (a volume locally, a PVC/Secret in k8s). The filter fails
# closed if the list is missing or stale.
ENV LADD_DIR=/data/ladd/active
USER nonroot:nonroot
ENTRYPOINT ["/filter"]
