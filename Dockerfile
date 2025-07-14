# -------------------- STAGE 1: Build Frontend --------------------
FROM oven/bun:latest AS builder

WORKDIR /build

# 复制 package.json 和锁文件
COPY web/package.json web/bun.lockb ./

# 使用 --frozen-lockfile 严格按照锁文件安装依赖
# 这可以极大地提高构建稳定性和速度
RUN bun install --frozen-lockfile

# 复制前端所有源代码
COPY ./web .

# 复制版本文件
COPY ./VERSION .

# 执行构建
# VITE_REACT_APP_VERSION 是一个自定义环境变量，确保你的代码里会使用它
# 如果你的代码使用的是 import.meta.env.VITE_APP_VERSION，则需要将变量名改为 VITE_APP_VERSION
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

# -------------------- STAGE 2: Build Go Backend --------------------
FROM golang:alpine AS builder2

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /build

# 优化Go的构建缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码（利用 .dockerignore 排除不必要文件）
COPY . .
# 从前端构建阶段复制编译好的静态文件
COPY --from=builder /build/dist ./web/dist

# 构建Go应用，并将版本号注入
RUN go build -ldflags "-s -w -X 'one-api/common.Version=$(cat VERSION)'" -o one-api

# -------------------- STAGE 3: Final Image --------------------
FROM alpine

# 更新证书并安装必要的运行时依赖 (tzdata 用于时区，ffmpeg 可能用于某些媒体处理)
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata ffmpeg \
    && update-ca-certificates

# 从Go构建阶段复制编译好的二进制文件
COPY --from=builder2 /build/one-api /

EXPOSE 3000


# 设置工作目录为数据目录
WORKDIR /data

# 启动程序
ENTRYPOINT ["/one-api"]
