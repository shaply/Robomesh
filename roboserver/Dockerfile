FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
RUN go build -o roboserver

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates bash && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/roboserver .
COPY --from=builder /app/config.yaml .

# handlers/ is mounted as a volume from the project root at runtime
RUN mkdir -p handlers

EXPOSE 8080 5000 1883

CMD ["./roboserver"]
