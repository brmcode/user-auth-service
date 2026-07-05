# Build stage
FROM golang:1.26-alpine3.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/http

# Run stage
FROM alpine:3.23
WORKDIR /app
RUN apk add --no-cache tzdata
COPY --from=builder /app/server .
COPY .env ./

ENV GIN_MODE=release

EXPOSE 8080
USER nobody
ENTRYPOINT ["./server"]
