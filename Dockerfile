FROM golang:alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && apk update && apk --no-cache add build-base
WORKDIR /go/src/github.com/qist/livetv/
COPY . . 
RUN GOPROXY="https://goproxy.io" GO111MODULE=on go build -o livetv .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata libc6-compat libgcc ffmpeg libstdc++ && \
    wget https://github.com/yt-dlp/yt-dlp/t/releases/latest/download/yt-dlp -O /usr/bin/yt-dlp && \
    chmod a+rx /usr/bin/yt-dlp
WORKDIR /root
COPY --from=builder /go/src/github.com/qist/livetv/view ./view
COPY --from=builder /go/src/github.com/qist/livetv/assert ./assert
COPY --from=builder /go/src/github.com/qist/livetv/.env .
COPY --from=builder /go/src/github.com/qist/livetv/livetv .
EXPOSE 9000
VOLUME ["/root/data"]
CMD ["./livetv"]