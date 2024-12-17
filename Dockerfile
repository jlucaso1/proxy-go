FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build

# Final stage
FROM scratch

WORKDIR /root/

# Copy the pre-built binary file
COPY --from=builder /app/proxy .

# Run the binary
CMD ["./proxy"]