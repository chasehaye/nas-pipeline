# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/database-writer ./cmd/database-writer

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/database-writer /database-writer
USER nonroot:nonroot
ENTRYPOINT ["/database-writer"]
