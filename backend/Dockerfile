# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Dipendenze Go in layer separato per cache ottimale
COPY go.mod go.sum ./
RUN go mod download

# Sorgente completo
COPY . .

# Build statico: CGO disabilitato per compatibilità con alpine minimal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /daydream ./cmd/server

# ── Stage 2: runtime ───────────────────────────────────────────────────────────
FROM alpine:3.20

# Certificati CA per le chiamate HTTPS all'API LLM
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Solo il binario e i file di configurazione statici necessari
COPY --from=builder /daydream .
COPY configs/ ./configs/

# Health check interno al container
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

EXPOSE 8080
ENTRYPOINT ["/app/daydream"]
