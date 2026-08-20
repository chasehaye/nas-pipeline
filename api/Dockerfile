# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
# Serves HTTP (HTTP_ADDR default :8090).
EXPOSE 8090
USER nonroot:nonroot
ENTRYPOINT ["/api"]
