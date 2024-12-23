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
    RUN CGO_ENABLED=0 GOOS=linux go build -o main .
    
    # ---------- Final Stage ----------
    FROM debian:stable-slim
    
    # Install necessary packages for Chromium
    RUN apt-get update && apt-get install -y --no-install-recommends \
        libnss3 \
        libatk-bridge2.0-0 \
        libgtk-3-0 \
        libx11-xcb1 \
        libxcomposite1 \
        libxcursor1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxi6 \
        libxrandr2 \
        libxrender1 \
        libxss1 \
        libxtst6 \
        fonts-liberation \
        libappindicator3-1 \
        libasound2 \
        chromium \
        ca-certificates \
        # Clean up apt caches to reduce image size
        && apt-get clean && rm -rf /var/lib/apt/lists/*
    
    # Set environment variables for Chromium
    ENV CHROMEDP_EXEC_PATH=/usr/bin/chromium
    
    # Set environment variable for GCP credentials
    ENV GOOGLE_APPLICATION_CREDENTIALS=/credentials/credentials.json
    
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