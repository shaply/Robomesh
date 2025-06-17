# Start with the official Golang image to build the app
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .

ENV GOOS=linux GOARCH=amd64
RUN go build -o roboserver

FROM debian:bookworm-slim
WORKDIR /app/
COPY --from=builder /app/roboserver .
CMD ["./roboserver"]
