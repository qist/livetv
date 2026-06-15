package service

import (
	"net/url"
	"path/filepath"
	"strings"
)

func splitGroupList(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "|", ",")
	s = strings.ReplaceAll(s, "\n", ",")
	parts := strings.Split(s, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func sanitizeM3UText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func tokenConfig() (enabled bool, param string, value string) {
	cc := GetCachedConfig()
	enabled = cc.TokenEnabled.Load()
	param, _ = cc.TokenParam.Load().(string)
	value, _ = cc.TokenValue.Load().(string)
	return
}

func BuildLiveM3U8URL(baseUrl string, channelParam string, channelID string) string {
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	path := baseUrl + "/live.m3u8"
	values := url.Values{}
	channelParam = strings.TrimSpace(channelParam)
	if channelParam == "" {
		channelParam = "c"
	}
	values.Set(channelParam, channelID)
	if enabled, param, value := tokenConfig(); enabled {
		values.Set(param, value)
	}
	return path + "?" + values.Encode()
}

func BuildTsProxyPrefix(baseUrl string) string {
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	path := baseUrl + "/live.ts"
	values := url.Values{}
	if enabled, param, value := tokenConfig(); enabled {
		values.Set(param, value)
	}
	encoded := values.Encode()
	if encoded == "" {
		return path + "?k="
	}
	return path + "?" + encoded + "&k="
}

func BuildPlaylistFileURL(baseUrl string, filename string) string {
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	filename = strings.TrimPrefix(filename, "/")
	path := baseUrl + "/" + filename
	values := url.Values{}
	if enabled, param, value := tokenConfig(); enabled {
		values.Set(param, value)
	}
	encoded := values.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func computeGroupTitles(groupName string, youtubeGroupTitles []string) []string {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = DefaultGroupName
	}
	if strings.EqualFold(groupName, "youtube") && len(youtubeGroupTitles) > 0 {
		return youtubeGroupTitles
	}
	return []string{groupName}
}

func NormalizePlaylistBaseName(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "/")
	v = filepath.Base(v)
	if v == "" || v == "." || v == "/" {
		return "lives"
	}
	ext := filepath.Ext(v)
	if ext != "" {
		v = strings.TrimSuffix(v, ext)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "lives"
	}
	return v
}

func DeriveM3UFilename(v string) string {
	return NormalizePlaylistBaseName(v) + ".m3u"
}

func DeriveTxtFilename(v string) string {
	return NormalizePlaylistBaseName(v) + ".txt"
}
