# Use an official Golang image as the build environment
FROM golang:1.20 as builder-server

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build the application
RUN go build -o echo-ip-server ./cmd/echo-ip

# Final stage: Run the binary in a minimal image
FROM debian:buster-slim as final-server

# Set environment variables for the server
ENV ECHO_IP_PORT=8745

# Copy the built binary and GeoLite2 database files from the builder stage
COPY --from=builder-server /app/echo-ip-server /usr/local/bin/echo-ip-server
COPY --from=builder-server /app/geolite /app/geolite

# Expose the application port
EXPOSE 8745

# Run the application
CMD ["echo-ip-server"]

# Use an official Golang image as the build environment
FROM golang:1.20 as builder-client

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build the CLI application
RUN go build -o echo-ip ./cmd/echo-ip

# Final stage: Run the binary in a minimal image
FROM debian:buster-slim as final-client

# Set environment variables for the client
ENV ECHO_IP_SERVICE_URL=https://example.com

# Copy the built binary from the builder stage
COPY --from=builder-client /app/echo-ip /usr/local/bin/echo-ip

# Set the entry point to the client binary
ENTRYPOINT ["echo-ip"]