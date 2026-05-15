# Lab quality Dockerfile. You will rewrite this in EP2 (multi stage, distroless, non root, .dockerignore, scanned).
FROM golang:1.26-alpine
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o /app/service .
CMD ["/app/service"]
