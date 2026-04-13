FROM golang:1.25 AS builder
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

# Run as non-root user to limit blast radius of container compromise
RUN useradd --system --no-create-home robomesh

COPY --from=builder /app/roboserver .
COPY --from=builder /app/config.yaml .

# handlers/ is mounted as a volume from the project root at runtime
RUN mkdir -p handlers && chown robomesh:robomesh handlers

USER robomesh

EXPOSE 8080 5002 5001/udp 1883

CMD ["./roboserver"]
