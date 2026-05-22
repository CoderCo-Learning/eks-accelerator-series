# syntax=docker/dockerfile:1.7
#
# Multi stage build for any of the Go services in platform/services/.
# Build from inside the service folder:
#   docker build -f ../../../02-containers/lab/Dockerfile -t order-service:slim .
#
# Or use the Makefile in this folder to build all nine.

# ---- builder ----
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache the dependency layer separately from the source.
COPY go.mod go.sum ./
RUN go mod download

# Now bring in the source and build a fully static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/service .

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/service /app/service

USER 65532:65532
# Default port for order-service. Override per service in compose / manifests.
EXPOSE 8081
ENTRYPOINT ["/app/service"]
