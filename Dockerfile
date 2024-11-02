# Use an official Golang image as the build environment
FROM golang:1.23 as builder-api

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build the application
RUN go build -o echo-ip-api .

# Final stage: Run the binary in a minimal image
FROM debian:buster-slim as final-api

# Set environment variables for the server
ENV ECHO_IP_PORT=8745

# Copy the built binary and GeoLite2 database files from the builder stage
COPY --from=builder-api /app/echo-ip-api /usr/local/bin/echo-ip-api
COPY --from=builder-api /app/geolite /app/geolite

# Expose the application port
EXPOSE 8745

# Run the application
CMD ["echo-ip-api"]

# Use an official Golang image as the build environment
FROM golang:1.23 as builder-client

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