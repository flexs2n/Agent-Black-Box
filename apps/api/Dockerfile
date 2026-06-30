# Stage 1: Build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/internal/migrate/migrations ./migrations

EXPOSE 4000

CMD ["./server"]