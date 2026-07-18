# ==========================
# Builder 阶段
# ==========================
# --platform=$BUILDPLATFORM：builder 始终运行在构建机原生架构上，
# 不走 QEMU/KVM 模拟，配合下面的 GOARCH 做交叉编译，大幅提升多架构构建速度。
FROM --platform=$BUILDPLATFORM golang:alpine AS builder
WORKDIR /go/src/github.com/qist/livetv/
ARG TARGETARCH
# 纯 Go 交叉编译：cgo 已禁用，Go 原生支持跨架构编译，无需目标架构 C 工具链。
COPY . .

# Go 环境配置：禁用 cgo 并按目标架构交叉编译
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=${TARGETARCH}
# 编译 livetv（在构建机原生架构上执行，输出目标架构二进制）
RUN go build -ldflags "-w -s" -trimpath -o livetv .

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
        unzip \
        nodejs \
        npm \
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

# 安装 deno (用于 yt-dlp EJS)
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        wget -O /tmp/deno.zip https://github.com/denoland/deno/releases/latest/download/deno-aarch64-unknown-linux-gnu.zip; \
    else \
        wget -O /tmp/deno.zip https://github.com/denoland/deno/releases/latest/download/deno-x86_64-unknown-linux-gnu.zip; \
    fi \
    && unzip -q /tmp/deno.zip -d /tmp \
    && mv /tmp/deno /usr/bin/deno \
    && chmod +x /usr/bin/deno \
    && rm -f /tmp/deno.zip

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
