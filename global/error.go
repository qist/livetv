package global

import "errors"

var (
	ErrConfigNotFound    = errors.New("config key not found")
	ErrYoutubeDlNotFound = errors.New("yt-dlp not found")
)
