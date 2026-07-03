# Stage 1: Build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./

RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite curl

WORKDIR /app

COPY --from=builder /app/apps/api/server .
COPY --from=builder /app/apps/api/internal/migrate/migrations ./migrations

EXPOSE 4000

CMD ["./server"]