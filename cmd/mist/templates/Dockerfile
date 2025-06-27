# Stage 1: Build the application
FROM golang:1.19-alpine AS build

# Install required build tools
RUN apk add --no-cache gcc musl-dev

# Set the working directory in the container
WORKDIR /app

# Copy go.mod and go.sum first (dependency caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o /{service-name} cmd/{service-name}/main.go

# Stage 2: Create a minimal runtime image
FROM alpine:3.17

# Install CA certificates to enable HTTPS
RUN apk add --no-cache ca-certificates

# Set the working directory in the container
WORKDIR /root/

# Copy the compiled binary from the build stage
COPY --from=build /app/main /{service-name}

# Expose port 80
EXPOSE 80

# Run the Go application
CMD ["./{service-name}"]
