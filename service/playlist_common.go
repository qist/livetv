package service

import (
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

func buildLiveM3U8URL(baseUrl string, channelParam string, channelID string) string {
	return baseUrl + "/live.m3u8?" + channelParam + "=" + channelID
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
