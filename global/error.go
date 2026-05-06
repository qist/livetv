package global

import "errors"

var (
	ErrConfigNotFound    = errors.New("config key not found")
	ErrYoutubeDlNotFound = errors.New("yt-dlp not found")
	ErrEmptyURL          = errors.New("empty url")
	ErrUpstreamStatus    = errors.New("upstream error status")
	ErrNotPlaylist       = errors.New("response is not a playlist")
)
