FROM golang:1.23 AS builder
WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -o e5-renewal-backend .

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/e5-renewal-backend /app/e5-renewal-backend

EXPOSE 8080
CMD ["/app/e5-renewal-backend"]
