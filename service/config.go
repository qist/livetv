package service

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qist/livetv/global"
	"github.com/qist/livetv/model"
)

// CachedConfig holds frequently-accessed config values to avoid repeated sync.Map lookups.
type CachedConfig struct {
	TokenEnabled atomic.Bool
	TokenParam   atomic.Value // string
	TokenValue   atomic.Value // string
	M3UFilename  atomic.Value // string
	TSTimeout    atomic.Int64 // duration in seconds
	TSCacheMB    atomic.Int64 // cache size in MB

	YtdlCmd     atomic.Value // string
	YtdlArgs    atomic.Value // string
	YtdlCookies atomic.Value // string
	YtdlTimeout atomic.Int64 // duration in seconds
	BaseUrl     atomic.Value // string
	ChannelParam atomic.Value // string
}

var cachedConfig CachedConfig

func init() {
	// Set defaults
	cachedConfig.TokenEnabled.Store(false)
	cachedConfig.TokenParam.Store("token")
	cachedConfig.TokenValue.Store("livetv")
	cachedConfig.M3UFilename.Store("lives")
	cachedConfig.TSTimeout.Store(30)
	cachedConfig.TSCacheMB.Store(200)
	cachedConfig.YtdlCmd.Store("yt-dlp")
	cachedConfig.YtdlArgs.Store("-f b -g {url}")
	cachedConfig.YtdlCookies.Store("")
	cachedConfig.YtdlTimeout.Store(20)
	cachedConfig.BaseUrl.Store("http://127.0.0.1:9000")
	cachedConfig.ChannelParam.Store("c")
}

// RefreshCachedConfig reloads all cached config values from the database/cache.
func RefreshCachedConfig() {
	if v, err := GetConfig("token_enabled"); err == nil {
		v = strings.TrimSpace(strings.ToLower(v))
		cachedConfig.TokenEnabled.Store(v == "1" || v == "true" || v == "yes" || v == "on")
	}
	if v, err := GetConfig("token_param"); err == nil {
		v = strings.TrimSpace(v)
		if v == "" {
			v = "token"
		}
		cachedConfig.TokenParam.Store(v)
	}
	if v, err := GetConfig("token_value"); err == nil {
		v = strings.TrimSpace(v)
		if v == "" {
			v = "livetv"
		}
		cachedConfig.TokenValue.Store(v)
	}
	if v, err := GetConfig("m3u_filename"); err == nil {
		cachedConfig.M3UFilename.Store(v)
	}
	if v, err := GetConfig("ts_timeout"); err == nil {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			cachedConfig.TSTimeout.Store(int64(sec))
		}
	}
	if v, err := GetConfig("ts_cache_max_size"); err == nil {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			cachedConfig.TSCacheMB.Store(int64(mb))
		}
	}
	if v, err := GetConfig("ytdl_cmd"); err == nil {
		cachedConfig.YtdlCmd.Store(v)
	}
	if v, err := GetConfig("ytdl_args"); err == nil {
		cachedConfig.YtdlArgs.Store(v)
	}
	if v, err := GetConfig("ytdl_cookies"); err == nil {
		cachedConfig.YtdlCookies.Store(v)
	}
	if v, err := GetConfig("ytdl_timeout"); err == nil {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			cachedConfig.YtdlTimeout.Store(int64(sec))
		}
	}
	if v, err := GetConfig("base_url"); err == nil {
		cachedConfig.BaseUrl.Store(v)
	}
	if v, err := GetConfig("channel_param"); err == nil {
		v = strings.TrimSpace(v)
		if v == "" {
			v = "c"
		}
		cachedConfig.ChannelParam.Store(v)
	}
}

func GetCachedConfig() *CachedConfig {
	return &cachedConfig
}

// GetTSTimeout returns the cached TS timeout duration.
func GetTSTimeout() time.Duration {
	return time.Duration(cachedConfig.TSTimeout.Load()) * time.Second
}

// GetTSCacheMaxBytes returns the cached TS cache max size in bytes.
func GetTSCacheMaxBytes() int64 {
	return cachedConfig.TSCacheMB.Load() * 1024 * 1024
}

// GetYtdlTimeout returns the cached yt-dlp timeout duration.
func GetYtdlTimeout() time.Duration {
	return time.Duration(cachedConfig.YtdlTimeout.Load()) * time.Second
}

func GetConfig(key string) (string, error) {
	if confValue, ok := global.ConfigCache.Load(key); ok {
		return confValue.(string), nil
	} else {
		var value model.Config
		err := global.DB.Where("name = ?", key).First(&value).Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return "", global.ErrConfigNotFound
			} else {
				return "", err
			}
		} else {
			global.ConfigCache.Store(key, value.Data)
			return value.Data, nil
		}
	}
}

func SetConfig(key, value string) error {
	data := model.Config{Name: key, Data: value}
	err := global.DB.Save(&data).Error
	if err == nil {
		global.ConfigCache.Store(key, value)
		// Refresh cached config values on any config change
		go RefreshCachedConfig()
	}
	return err
}
