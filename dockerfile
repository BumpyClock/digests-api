# Start from the latest golang base image
FROM golang:latest

# Add Maintainer Info
LABEL maintainer="Aditya Sharma <aditya@adityasharma.net>"

# Install necessary packages
RUN apt-get update && apt-get install -y \
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
    chromium

# Set environment variables for Chromium
ENV CHROMEDP_EXEC_PATH=/usr/bin/chromium

ENV GOOGLE_APPLICATION_CREDENTIALS=/credentials/credentials.json


# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY src/go.mod src/go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY src/ .

# Build the Go app
RUN GOOS=linux GOARCH=amd64 go build -o main .

# Expose port 8080 to the outside world
EXPOSE 8000

# Command to run the executable
CMD ["./main", "--redis", "redis:6379"]