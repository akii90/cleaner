# Build Stage
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o pod-cleaner cmd/main.go

# Runtime Stage
FROM alpine:3.23.3

WORKDIR /
COPY --from=builder /app/pod-cleaner .

ENTRYPOINT ["/pod-cleaner"]
