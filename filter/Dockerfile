# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
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
