# ---- Stage 1: Build frontend ----
FROM node:22-alpine AS frontend
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: Build Go binary ----
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# 将前端构建产物放到 embed 目标路径
COPY --from=frontend /app/dist ./static/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /e5-renewal .

# ---- Stage 3: Final minimal image ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /e5-renewal /e5-renewal
EXPOSE 8080
ENTRYPOINT ["/e5-renewal"]
