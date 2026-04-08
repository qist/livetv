package global

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

var defaultConfigValue = map[string]string{
	"ytdl_cmd":            "yt-dlp",
	"ytdl_args":           "-f b -g {url}",
	"ytdl_cookies":        "",
	"ytdl_cookies_domain": ".youtube.com",
	"ytdl_timeout":        "20",
	"base_url":            "http://127.0.0.1:9000",
	"password":            "password",
	"m3u_filename":        "lives",
	"channel_param":       "c",
	"ts_timeout":          "30",
	"youtube_m3u_groups":  "YouTube",
}

var (
	HttpClientTimeout = 30 * time.Second
	ConfigCache       sync.Map
	URLCache          sync.Map
	M3U8Cache         = cache.New(3*time.Second, 10*time.Second)
	HTTPTransport     = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   128,
		MaxConnsPerHost:       256,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	PlaylistHTTPClient = &http.Client{
		Timeout:   HttpClientTimeout,
		Transport: HTTPTransport,
	}
	StreamHTTPClient = &http.Client{
		Transport: HTTPTransport,
	}
	IOBufferPool = sync.Pool{
		New: func() any {
			return make([]byte, 32*1024)
		},
	}
)
