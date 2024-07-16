# Build stage
FROM golang:1.20.14-bullseye AS builder

# Set environment variables for Go
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /
COPY . /code
RUN cd /code && make all
RUN cd /code && make release-ran-simulator && cp ransim-latest.tar.gz /ransim-latest.tar.gz


# Final stage
FROM debian:bookworm-slim AS final

COPY --from=builder /ransim-latest.tar.gz .
RUN tar zxvf ransim-latest.tar.gz
WORKDIR /ran-simulator

# Create a non-root user 'gouser'
RUN useradd --create-home --shell /bin/bash gouser
RUN chown -R gouser:gouser /ran-simulator
USER gouser

# Set the entrypoint
ENTRYPOINT ["./ransim"]
