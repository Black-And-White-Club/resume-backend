# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy only the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of your application source code
COPY . .

# Set the Go build cache directory
ENV GOCACHE=/root/.cache/go-build

# Enable caching for the Go build process and specify the output binary path
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /main/app .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Install runtime dependencies for SQLite (if needed)
RUN apk --no-cache add sqlite-libs

# Copy the executable from the builder stage
COPY --from=builder /main/app /main/app

# Expose port 8000
EXPOSE 8000

# Set the command to run the executable
ENTRYPOINT ["/main/app"]
