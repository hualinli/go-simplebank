# Build Stage
FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o main ./cmd/main.go

# Final Stage
FROM alpine:3.23
WORKDIR /app
COPY --from=builder /app/main .


EXPOSE 8080
CMD ["./main"]
