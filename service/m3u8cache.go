package service

import (
	"log"
	"regexp"
	"time"

	"github.com/qist/livetv/global"
	"github.com/qist/livetv/util"
)

// Channel failure tracking
var (
	channelFailures = make(map[string]int)
	failedChannels  = make(map[string]time.Time)
)

const (
	maxFailures     = 3
	failureCooldown = 12 * time.Hour
)

func LoadChannelCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	for _, v := range channels {
		channelURL := normalizeYoutubeURL(v.URL)
		log.Println("caching", channelURL)
		_, err := GetYoutubeLiveM3U8(channelURL)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(channelURL, "cached")
	}
}

func UpdateURLCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	for _, v := range channels {
		channelURL := normalizeYoutubeURL(v.URL)
		if failTime, failed := failedChannels[channelURL]; failed {
			if time.Since(failTime) < failureCooldown {
				log.Println("skipping failed channel (cooldown):", channelURL)
				continue
			}
			delete(failedChannels, channelURL)
			delete(channelFailures, channelURL)
		}

		log.Println("caching", channelURL)
		_, err := GetYoutubeLiveM3U8(channelURL)
		if err != nil {
			log.Println(err)
			channelFailures[channelURL]++
			log.Println("channel failure count:", channelURL, channelFailures[channelURL])

			if channelFailures[channelURL] >= maxFailures {
				log.Println("channel failed too many times, putting in cooldown:", channelURL)
				failedChannels[channelURL] = time.Now()
			}
		} else {
			delete(channelFailures, channelURL)
			delete(failedChannels, channelURL)
			log.Println(channelURL, "cached")
		}
	}
	global.URLCache.Range(func(k, v any) bool {
		value := v.(string)
		regex := regexp.MustCompile(`/expire/(\d+)/`)
		matched := regex.FindStringSubmatch(value)
		if len(matched) < 2 {
			global.URLCache.Delete(k)
		}
		expireTime := time.Unix(util.String2Int64(matched[1]), 0)
		if time.Now().After(expireTime) {
			global.URLCache.Delete(k)
		}
		return true
	})
}

func ResetYtdlCaches() {
	global.URLCache.Range(func(k, v any) bool {
		global.URLCache.Delete(k)
		return true
	})
	channelFailures = make(map[string]int)
	failedChannels = make(map[string]time.Time)
	resetYtdlFailureState()
}
