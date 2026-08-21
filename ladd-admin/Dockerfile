# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -p=1 -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd/server

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENV LADD_ADMIN_IDENTITY_PATH=/keys/identity.txt
USER nonroot:nonroot
ENTRYPOINT ["/server"]
