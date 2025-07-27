# Build Stage
FROM golang:1.24-alpine3.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./main.go

# Run Stage
FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .
COPY wait-for.sh .

ENV GIN_MODE=release

EXPOSE 8080
CMD ["/app/main"]
