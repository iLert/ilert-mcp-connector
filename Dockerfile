# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o ilert-mcp-connector .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/ilert-mcp-connector .

EXPOSE 8383

CMD ["./ilert-mcp-connector"]

