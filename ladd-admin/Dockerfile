# syntax=docker/dockerfile:1

# ---- build ----
# Build context is the REPO ROOT so the shared platform module is available.
# Build with:  docker build -f ladd-admin/Dockerfile .
FROM golang:1.26.1 AS build
WORKDIR /src
COPY platform/ ./platform/
COPY ladd-admin/go.mod ladd-admin/go.sum ./ladd-admin/
RUN cd ladd-admin && go mod download
COPY ladd-admin/ ./ladd-admin/
RUN cd ladd-admin && CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd/server

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENV LADD_ADMIN_IDENTITY_PATH=/keys/identity.txt
USER nonroot:nonroot
ENTRYPOINT ["/server"]
