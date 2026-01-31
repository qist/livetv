# ==========================
# Builder 阶段
# ==========================
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

# 安装构建依赖
RUN apk add --no-cache build-base git

WORKDIR /go/src/github.com/qist/livetv/
COPY . .

# Go 环境配置
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=latest
ENV GOPROXY="https://goproxy.io"
ENV GO111MODULE=on
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
# 编译 livetv
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o livetv .

# ==========================
# Runtime 阶段
# ==========================
FROM alpine:latest

# 安装运行依赖
RUN set -ex \
    && apk --no-cache add \
        ca-certificates \
        tzdata \
        ffmpeg \
    && update-ca-certificates

# 设置工作目录
WORKDIR /root

# 多平台 yt-dlp 下载
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        wget -O /usr/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_musllinux_aarch64; \
    else \
        wget -O /usr/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_musllinux; \
    fi \
    && chmod +x /usr/bin/yt-dlp

# 拷贝 Go 服务文件
COPY --from=builder /go/src/github.com/qist/livetv/view ./view
COPY --from=builder /go/src/github.com/qist/livetv/assert ./assert
COPY --from=builder /go/src/github.com/qist/livetv/.env .
COPY --from=builder /go/src/github.com/qist/livetv/livetv .

# 暴露端口和挂载卷
EXPOSE 9000
VOLUME ["/root/data"]

# 启动服务
CMD ["./livetv"]
