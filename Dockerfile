# Web build stage
FROM node:18-alpine as web-builder

WORKDIR /app
COPY seanime-web/package*.json ./
RUN npm install && \
    npm install -g env-cmd && \
    rm -rf /root/.npm

COPY seanime-web/ ./

# Create production env file
RUN echo "NEXT_PUBLIC_APP_VERSION=1.0.0\n\
NEXT_PUBLIC_API_URL=http://localhost:43211\n\
NEXT_PUBLIC_BASE_URL=http://localhost:43210\n\
NEXT_PUBLIC_NODE_ENV=production\n\
NEXT_PUBLIC_PLATFORM=web" > .env.web && \
    NODE_ENV=production env-cmd -f .env.web npm run build && \
    mkdir -p /app/web && \
    cp -r out/* /app/web/ && \
    rm -rf node_modules .next

# Go build stage
FROM golang:1.23-alpine as go-builder

WORKDIR /app
COPY go.* ./
ENV GOTOOLCHAIN=auto GOOS=linux

# Install build dependencies and build in a single layer
RUN apk add --no-cache gcc musl-dev && \
    go mod download && \
    rm -rf /root/.cache/go-build

COPY . .
COPY --from=web-builder /app/web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o seanime main.go

# Final stage
FROM ubuntu:22.04

LABEL maintainer="animechanica"
LABEL description="Container for Seanime with qBittorrent and MPV"

# Install essential packages and clean up in a single layer
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    curl \
    wget \
    git \
    unzip \
    python3-minimal \
    python3-pip \
    mpv \
    qbittorrent-nox \
    ca-certificates \
    tzdata \
    netcat \
    dnsutils \
    procps && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /data /media/anime /media/dl_anime /media/dl_manga /media/manga /root/.config/qBittorrent

# Set environment variables
ENV HOME="/root" \
    CONFIG_DIR="/data" \
    ANIME_DIR="/media/anime" \
    MANGA_DIR="/media/manga" \
    DL_ANIME_DIR="/media/dl_anime" \
    DL_MANGA_DIR="/media/dl_manga" \
    PATH="/usr/local/bin:${PATH}"

# Copy built binary and configuration files
COPY --from=go-builder /app/seanime /usr/local/bin/seanime
COPY entrypoint.sh /entrypoint.sh
COPY qbittorrent-setup.sh /usr/local/bin/qbittorrent-setup.sh
COPY tls-patch.sh /usr/local/bin/tls-patch.sh

# Set permissions
RUN chmod +x /entrypoint.sh /usr/local/bin/qbittorrent-setup.sh /usr/local/bin/tls-patch.sh

# Expose ports
EXPOSE 43211 8085

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:43211/ || exit 1

ENTRYPOINT ["/entrypoint.sh"]
