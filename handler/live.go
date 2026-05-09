package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/qist/livetv/global"
	"github.com/qist/livetv/service"
	"github.com/qist/livetv/util"
)

var hopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func M3UHandler(c *gin.Context) {
	content, err := service.M3UGenerate()
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Status(http.StatusOK)
	_, _ = io.WriteString(c.Writer, content)
}

func TxtHandler(c *gin.Context) {
	content, err := service.TxtGenerate()
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK)
	_, _ = io.WriteString(c.Writer, content)
}

func LiveHandler(c *gin.Context) {
	channelParam, err := service.GetConfig("channel_param")
	if err != nil {
		channelParam = "c"
	}
	channelIdentifier := c.Query(channelParam)
	if channelIdentifier == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	channelInfo, err := service.GetChannel(channelIdentifier)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}
	m3u8Body, err := service.FetchChannelPlaylist(channelInfo.URL, channelInfo.Proxy)
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Status(http.StatusOK)
	_, _ = io.WriteString(c.Writer, m3u8Body)
}

func TsProxyHandler(c *gin.Context) {
	zipedRemoteURL := c.Query("k")
	remoteURL, err := util.DecompressString(zipedRemoteURL)
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if remoteURL == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 读取并更新缓存大小配置（MB）
	service.GlobalTSCache.UpdateMaxBytes(service.GetTSCacheMaxBytes())

	tsCache := service.GlobalTSCache
	item, created := tsCache.GetOrCreate(zipedRemoteURL)

	if !created {
		if item.IsSealed() {
			log.Println("[TS] cache hit:", remoteURL)
		} else {
			log.Println("[TS] join stream:", remoteURL)
		}
		done := c.Request.Context().Done()
		writeStreamHeaders(c, nil, 0)
		if err := item.ReadAll(c.Writer, done); err != nil && !errors.Is(err, context.Canceled) {
			log.Println("[TS] read error:", err)
		}
		return
	}

	// 新建缓存项，从源拉取并流式写入
	log.Println("[TS] fetch start:", remoteURL)
	timeout := getTSTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := doFetchTSRemote(ctx, c.Request.Method, remoteURL, c.Request.Header)
	if err != nil {
		log.Println("[TS] fetch error:", err)
		tsCache.Remove(zipedRemoteURL)
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 写响应头（使用上游响应头）
	writeStreamHeaders(c, resp.Header, resp.StatusCode)

	// 流式读取上游，同时写入缓存和客户端
	total := 0
	buf := global.IOBufferPool.Get().([]byte)
	defer global.IOBufferPool.Put(buf)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			tsCache.WriteChunk(item, chunk)
			if _, writeErr := c.Writer.Write(chunk); writeErr != nil {
				if !errors.Is(writeErr, context.Canceled) {
					log.Println("[TS] write error:", writeErr)
				}
				item.Seal(nil)
				return
			}
			total += n
		}
		if readErr != nil {
			if readErr == io.EOF {
				item.Seal(nil)
			} else {
				item.Seal(readErr)
			}
			break
		}
	}
	log.Printf("[TS] streamed %dKB: %s\n", total/1024, remoteURL)
}

func writeStreamHeaders(c *gin.Context, upstreamHeader http.Header, statusCode int) {
	if upstreamHeader != nil {
		copyHeaders(c.Writer.Header(), upstreamHeader)
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("X-Accel-Buffering", "no")
	if statusCode > 0 {
		c.Status(statusCode)
	} else {
		c.Status(http.StatusOK)
	}
}

func getTSTimeout() time.Duration {
	return service.GetTSTimeout()
}

func doFetchTSRemote(ctx context.Context, method, remoteURL string, reqHeaders http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, remoteURL, nil)
	if err != nil {
		return nil, err
	}
	copyHeaders(req.Header, reqHeaders)
	if ua := reqHeaders.Get("User-Agent"); ua != "" {
		req.Header.Set("User-Agent", ua)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}
	return global.StreamHTTPClient.Do(req)
}

func CacheHandler(c *gin.Context) {
	var sb strings.Builder
	global.URLCache.Range(func(k, v any) bool {
		sb.WriteString(k.(string))
		sb.WriteString(" => ")
		sb.WriteString(v.(string))
		sb.WriteString("\n")
		return true
	})
	c.Data(http.StatusOK, "text/plain", []byte(sb.String()))
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if _, skip := hopHeaders[http.CanonicalHeaderKey(key)]; skip {
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
