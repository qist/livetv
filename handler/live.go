package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
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

func LiveHandler(c *gin.Context) {
	var m3u8Body string
	channelParam, err := service.GetConfig("channel_param")
	if err != nil {
		channelParam = "c"
	}
	channelCacheKey := c.Query(channelParam)
	iBody, found := global.M3U8Cache.Get(channelCacheKey)
	if found {
		m3u8Body = iBody.(string)
	} else {
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
		baseUrl, err := service.GetConfig("base_url")
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		liveM3U8, err := service.GetYoutubeLiveM3U8(channelInfo.URL)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		resp, err := global.PlaylistHTTPClient.Get(liveM3U8)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			log.Println("upstream status:", resp.Status)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.HasPrefix(contentType, "video/") && !strings.Contains(contentType, "mpegurl") {
			// Likely VOD media (mp4/flv), not a live HLS playlist.
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		const maxM3U8Size = 2 * 1024 * 1024 // 2MB hard limit for playlists
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxM3U8Size))
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		bodyString := string(bodyBytes)
		if !strings.HasPrefix(strings.TrimSpace(bodyString), "#EXTM3U") {
			// Not a playlist; avoid caching large VOD responses.
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if channelInfo.Proxy {
			m3u8Body = service.M3U8Process(bodyString, baseUrl+"/live.ts?k=")
		} else {
			m3u8Body = bodyString
		}
		global.M3U8Cache.Set(channelCacheKey, m3u8Body, 3*time.Second)
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
	timeout := global.HttpClientTimeout
	if cfgTimeout, configErr := service.GetConfig("ts_timeout"); configErr == nil {
		if sec, parseErr := strconv.Atoi(cfgTimeout); parseErr == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, remoteURL, nil)
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	copyHeaders(req.Header, c.Request.Header)
	resp, err := global.StreamHTTPClient.Do(req)
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	copyHeaders(c.Writer.Header(), resp.Header)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("X-Accel-Buffering", "no")
	c.Status(resp.StatusCode)
	buffer := global.IOBufferPool.Get().([]byte)
	defer global.IOBufferPool.Put(buffer)
	if _, err := io.CopyBuffer(c.Writer, resp.Body, buffer); err != nil && !errors.Is(err, context.Canceled) {
		log.Println(err)
	}
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
