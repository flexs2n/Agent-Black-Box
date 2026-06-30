# Stage 1: Build
FROM golang:1.23-alpine AS builder

ENV GONOSUMCHECK=*
ENV GONOSUMDB=*
ENV GOFLAGS=-mod=mod

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/internal/migrate/migrations ./migrations

EXPOSE 4000

CMD ["./server"]