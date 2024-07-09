# Build stage
FROM golang:alpine3.19 AS builder

# Set environment variables for Go
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create and set the working directory
WORKDIR /code

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o ransim ./cmd/ransim

# Final stage
FROM alpine:3.19

# Create and set the working directory
WORKDIR /code

# Copy the binary from the builder stage
COPY --from=builder /code/ransim .

# Create a non-root user 'gouser'
RUN useradd --create-home --shell /bin/bash gouser
RUN chown -R gouser:gouser /code
USER gouser

# Set the entrypoint
ENTRYPOINT ["./ransim"]
