# Dockerfile — Go CLI + Python AI Brain (full image)
#
# Build:  docker build -t probex:latest -f docker/Dockerfile .
# Run:    docker run probex:latest scan https://api.example.com
# Serve:  docker run -p 9712:9712 probex:latest serve --listen 0.0.0.0:9712

# Stage 1: Build Go binary
FROM golang:1.23-alpine AS go-builder
WORKDIR /build
COPY core/go.mod core/go.sum ./
RUN go mod download
COPY core/ .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /probex .

# Stage 2: Python brain + Go binary
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Install Go binary
COPY --from=go-builder /probex /usr/local/bin/probex

# Install Python brain
WORKDIR /app
COPY brain/pyproject.toml brain/pyproject.toml
COPY brain/src/ brain/src/
RUN pip install --no-cache-dir ./brain

EXPOSE 9712

# Default: run the Go CLI (which starts Python brain as needed via --ai)
ENTRYPOINT ["probex"]
