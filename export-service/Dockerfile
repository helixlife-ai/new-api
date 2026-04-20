# Export Service Dockerfile
# 多阶段构建：前端 -> Go 后端 -> 最终镜像

# ==================== 前端构建阶段 ====================
FROM node:18-alpine AS web-builder

WORKDIR /build/web
COPY web/package.json .
RUN npm install
COPY web/ .
RUN npm run build

# ==================== Go 构建阶段 ====================
FROM golang:1.23-alpine AS builder

WORKDIR /build

# 先复制依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

# 再复制源代码
COPY config/ ./config/
COPY controller/ ./controller/
COPY model/ ./model/
COPY router/ ./router/
COPY service/ ./service/
COPY main.go .

# 复制前端构建产物
COPY --from=web-builder /build/web/dist ./web/dist

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o export-service .

# ==================== 运行阶段 ====================
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 复制二进制文件
COPY --from=builder /build/export-service .

# 复制前端静态文件（供运行时访问）
COPY --from=builder /build/web/dist ./web/dist

EXPOSE 8080

ENTRYPOINT ["./export-service"]
