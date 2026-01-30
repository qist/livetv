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
		log.Println("caching", v.URL)
		liveURL, err := RealGetYoutubeLiveM3U8(v.URL)
		if err != nil {
			log.Println(err)
			// Don't return on single channel failure
			continue
		}
		global.URLCache.Store(v.URL, liveURL)
		log.Println(v.URL, "cached")
	}
}

func UpdateURLCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	for _, v := range channels {
		// Check if channel is in cooldown period
		if failTime, failed := failedChannels[v.URL]; failed {
			if time.Since(failTime) < failureCooldown {
				log.Println("skipping failed channel (cooldown):", v.URL)
				continue
			}
			// Cooldown expired, remove from failed list
			delete(failedChannels, v.URL)
			delete(channelFailures, v.URL)
		}

		log.Println("caching", v.URL)
		liveURL, err := RealGetYoutubeLiveM3U8(v.URL)
		if err != nil {
			log.Println(err)
			// Track failure
			channelFailures[v.URL]++
			log.Println("channel failure count:", v.URL, channelFailures[v.URL])

			// If max failures reached, put in cooldown
			if channelFailures[v.URL] >= maxFailures {
				log.Println("channel failed too many times, putting in cooldown:", v.URL)
				failedChannels[v.URL] = time.Now()
			}
		} else {
			// Success, reset failure count
			delete(channelFailures, v.URL)
			delete(failedChannels, v.URL)

			global.URLCache.Store(v.URL, liveURL)
			log.Println(v.URL, "cached")
		}
	}
	global.URLCache.Range(func(k, v interface{}) bool {
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
