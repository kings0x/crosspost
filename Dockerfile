# ── Build stage ──────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

#copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api/main.go


# ── Final stage ──────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# needed for HTTPS calls to external services (infisical, render etc)
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/api .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./api"]