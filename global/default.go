package global

import (
	"net"
	"net/http"
	"os"
	"strings"
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
	"token_enabled":       "0",
	"token_param":         "token",
	"token_value":         "livetv",
	"ts_cache_max_size":   "200",
}

var (
	HttpClientTimeout = 30 * time.Second
	ConfigCache       sync.Map
	URLCache          sync.Map
	M3U8Cache         = cache.New(3*time.Second, 10*time.Second)
	// HTTPDialer is the shared dialer for all outbound HTTP connections.
	// CVE-2026-98432: MPTCP is disabled by default; see MPTCPEnabled.
	HTTPDialer = &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	HTTPTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           HTTPDialer.DialContext,
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

// MPTCPEnabled reports whether Multipath TCP may be used for new
// connections. CVE-2026-98432: MPTCP is disabled by default; it can only be
// opted into explicitly with LIVETV_MPTCP=1.
func MPTCPEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LIVETV_MPTCP"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
