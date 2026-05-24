# ==============================
# Stage 1: 构建前端
# ==============================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ==============================
# Stage 2: 构建后端
# ==============================
FROM golang:1.22-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o modelgate-server .

# ==============================
# Stage 3: 运行时
# ==============================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

# 复制后端 binary
COPY --from=backend-builder /app/modelgate-server .

# 复制前端产物
COPY --from=frontend-builder /app/web/dist ./web/dist

# 复制配置模板（实际配置通过 volume 挂载或环境变量覆盖）
COPY config.yaml ./config.yaml

EXPOSE 8080

CMD ["./modelgate-server", "-config", "config.yaml"]
