FROM golang:1.26.1-alpine
WORKDIR /app
COPY . .
RUN go build -o main ./cmd

EXPOSE 8080
CMD ["./main"]