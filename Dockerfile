# Build Stage
FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd

# Final Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .


EXPOSE 8080
CMD ["./main"]