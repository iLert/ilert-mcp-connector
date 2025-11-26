# Build stage
FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

ARG VERSION=dev
ARG COMMIT=unknown

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o ilert-mcp-connector .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/ilert-mcp-connector .

EXPOSE 8383

CMD ["./ilert-mcp-connector"]

