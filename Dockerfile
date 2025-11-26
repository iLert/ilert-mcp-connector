# Build stage
FROM golang:1.25 AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

ARG VERSION=dev
ARG COMMIT=unknown

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o ilert-mcp-connector .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /app/ilert-mcp-connector .

EXPOSE 8383

CMD ["./ilert-mcp-connector"]

