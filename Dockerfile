# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod tidy && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o server ./cmd/server

# Copy templates
COPY templates/ /app/templates/

# Runtime stage - use Debian for LaTeX support
FROM debian:bookworm-slim
WORKDIR /app

# Install LaTeX and dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    texlive-latex-base \
    texlive-latex-extra \
    texlive-fonts-recommended \
    zip \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && addgroup --gid 1000 appuser \
    && adduser --uid 1000 --gid 1000 --disabled-password --gecos "" appuser

COPY --from=builder /app/server .
COPY --from=builder /app/templates /app/templates
RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8086
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8086/health || exit 1

CMD ["./server"]
