package service

import (
	"log"
	"strconv"
	"strings"
)

func M3UGenerate() (string, error) {
	baseUrl, err := GetConfig("base_url")
	if err != nil {
		log.Println(err)
		return "", err
	}
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return "", err
	}
	var m3u strings.Builder
	m3u.WriteString("#EXTM3U\n")
	channelParam, err := GetConfig("channel_param")
	if err != nil {
		channelParam = "c"
	}
	youtubeM3UGroups, err := GetConfig("youtube_m3u_groups")
	if err != nil || strings.TrimSpace(youtubeM3UGroups) == "" {
		youtubeM3UGroups = "YouTube"
	}
	youtubeGroupTitles := splitGroupList(youtubeM3UGroups)
	for _, v := range channels {
		channelID := strconv.Itoa(int(v.ID))
		if v.CustomID != "" {
			channelID = v.CustomID
		}
		displayName := sanitizeM3UText(v.Name)
		groupTitles := computeGroupTitles(v.GroupName, youtubeGroupTitles)
		for _, groupTitle := range groupTitles {
			groupTitle = sanitizeM3UText(groupTitle)
			m3u.WriteString("#EXTINF:-1")
			m3u.WriteString(" tvg-id=\"")
			m3u.WriteString(escapeM3UAttrValue(channelID))
			m3u.WriteString("\"")
			m3u.WriteString(" tvg-name=\"")
			m3u.WriteString(escapeM3UAttrValue(displayName))
			m3u.WriteString("\"")
			m3u.WriteString(" group-title=\"")
			m3u.WriteString(escapeM3UAttrValue(groupTitle))
			m3u.WriteString("\"")
			m3u.WriteString(",")
			m3u.WriteString(displayName)
			m3u.WriteString("\n")
			m3u.WriteString(buildLiveM3U8URL(baseUrl, channelParam, channelID))
			m3u.WriteString("\n")
		}
	}
	return m3u.String(), nil
}

func escapeM3UAttrValue(s string) string {
	s = sanitizeM3UText(s)
	return strings.ReplaceAll(s, "\"", "'")
}
