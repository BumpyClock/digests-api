# Start from the latest golang base image
FROM golang:latest

# Add Maintainer Info
LABEL maintainer="Aditya Sharma <aditya@adityasharma.net>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY src/go.mod src/go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY src/ .

# Build the Go app
# RUN go build -o main .
RUN GOOS=linux GOARCH=amd64 go build -o main .

# Expose port 8080 to the outside world
EXPOSE 8000

# Command to run the executable
CMD ["./main", "--redis", "redis:6379"]