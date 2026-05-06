package service

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/qist/livetv/global"
)

type playlistCall struct {
	done     chan struct{}
	m3u8Body string
	err      error
}

var (
	playlistCallMu   sync.Mutex
	playlistInflight = make(map[string]*playlistCall)
)

func FetchChannelPlaylist(youtubeURL string, proxy bool) (string, error) {
	cacheKey := NormalizeYoutubeURL(youtubeURL)
	if cacheKey == "" {
		return "", global.ErrEmptyURL
	}

	// 缓存原始 M3U8，读取时按需转换
	if body, ok := global.M3U8Cache.Get(cacheKey); ok {
		return applyProxy(body.(string), proxy)
	}

	call, waiting := getOrCreatePlaylistCall(cacheKey)
	if waiting {
		<-call.done
		return applyProxy(call.m3u8Body, call.err == nil && proxy)
	}
	defer finishPlaylistCall(cacheKey, call)

	if body, ok := global.M3U8Cache.Get(cacheKey); ok {
		call.m3u8Body = body.(string)
		return applyProxy(call.m3u8Body, proxy)
	}

	liveM3U8, err := GetYoutubeLiveM3U8(cacheKey)
	if err != nil {
		call.err = err
		return "", err
	}

	// 始终拉取原始 M3U8 缓存
	m3u8Body, err := fetchRawM3U8(liveM3U8)
	if err != nil {
		call.err = err
		return "", err
	}

	call.m3u8Body = m3u8Body
	global.M3U8Cache.Set(cacheKey, m3u8Body, 3*time.Second)
	return applyProxy(m3u8Body, proxy)
}

// applyProxy 按需转换 M3U8：proxy=true 时替换 TS URL，否则返回原始内容
func applyProxy(body string, proxy bool) (string, error) {
	if !proxy {
		return body, nil
	}
	baseUrl, err := GetConfig("base_url")
	if err != nil {
		return "", err
	}
	return M3U8Process(body, BuildTsProxyPrefix(baseUrl)), nil
}

func fetchRawM3U8(liveM3U8 string) (string, error) {
	resp, err := global.PlaylistHTTPClient.Get(liveM3U8)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		log.Println("upstream status:", resp.Status)
		return "", global.ErrUpstreamStatus
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "video/") && !strings.Contains(contentType, "mpegurl") {
		return "", global.ErrNotPlaylist
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}
	bodyString := string(bodyBytes)
	if !strings.HasPrefix(strings.TrimSpace(bodyString), "#EXTM3U") {
		return "", global.ErrNotPlaylist
	}
	return bodyString, nil
}

func getOrCreatePlaylistCall(cacheKey string) (*playlistCall, bool) {
	playlistCallMu.Lock()
	defer playlistCallMu.Unlock()
	if call, ok := playlistInflight[cacheKey]; ok {
		return call, true
	}
	call := &playlistCall{done: make(chan struct{})}
	playlistInflight[cacheKey] = call
	return call, false
}

func finishPlaylistCall(cacheKey string, call *playlistCall) {
	playlistCallMu.Lock()
	delete(playlistInflight, cacheKey)
	playlistCallMu.Unlock()
	close(call.done)
}
