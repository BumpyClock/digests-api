# ---------- Builder Stage ----------
    FROM golang:1.23 AS builder

    # Create and switch to the /app directory
    WORKDIR /app
    
    # Copy go.mod and go.sum first and download dependencies
    COPY src/go.mod src/go.sum ./
    RUN go mod download
    
    # Copy the remaining source code
    COPY src/ .
    
    # Build the Go app (static binary if possible)
    RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api
    
    # ---------- Final Stage ----------
   # Start a new stage from scratch
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates
   

    
    # Create and switch to the /app directory
    WORKDIR /app
    
    # Copy only the compiled binary from the builder stage
    COPY --from=builder /app/main .
    
    # If you need credentials.json inside the container, copy it in.
    # (If you mount credentials in production, you can remove this line.)
    # COPY src/credentials.json /credentials/credentials.json
    
    # Expose port 8000
    EXPOSE 8000
    
    # Command to run the executable
    CMD ["./main", "--redis", "redis:6379"]