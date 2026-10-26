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
# 注意：必须使用 Alpine 仓库中的 deno（musl 构建）。官方 GitHub release 的
# deno-*-unknown-linux-gnu.zip 依赖 glibc，在 Alpine 上无法执行，会导致
# yt-dlp 报 "No supported JavaScript runtime could be found"。
RUN set -ex \
    && apk --no-cache add \
        ca-certificates \
        tzdata \
        ffmpeg \
        deno \
        nodejs \
        npm \
    && update-ca-certificates \
    && deno --version

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
