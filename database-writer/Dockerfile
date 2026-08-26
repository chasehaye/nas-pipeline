# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f database-writer/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY database-writer/go.mod database-writer/go.sum ./database-writer/
RUN cd database-writer && go mod download
COPY database-writer/ ./database-writer/
RUN cd database-writer && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/database-writer ./cmd/database-writer

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/database-writer /database-writer
USER nonroot:nonroot
ENTRYPOINT ["/database-writer"]
