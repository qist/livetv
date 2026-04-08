package service

import (
	"log"
	"strconv"
	"strings"
)

func TxtGenerate() (string, error) {
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
	channelParam, err := GetConfig("channel_param")
	if err != nil {
		channelParam = "c"
	}
	youtubeM3UGroups, err := GetConfig("youtube_m3u_groups")
	if err != nil || strings.TrimSpace(youtubeM3UGroups) == "" {
		youtubeM3UGroups = "YouTube"
	}
	youtubeGroupTitles := splitGroupList(youtubeM3UGroups)

	type entry struct {
		name string
		url  string
	}

	groupOrder := make([]string, 0, 16)
	groupEntries := make(map[string][]entry, 16)

	addGroup := func(group string) {
		if _, ok := groupEntries[group]; ok {
			return
		}
		groupEntries[group] = nil
		groupOrder = append(groupOrder, group)
	}

	for _, v := range channels {
		channelID := strconv.Itoa(int(v.ID))
		if v.CustomID != "" {
			channelID = v.CustomID
		}
		name := sanitizeTxtField(v.Name)
		groupTitles := computeGroupTitles(v.GroupName, youtubeGroupTitles)
		url := buildLiveM3U8URL(baseUrl, channelParam, channelID)
		for _, groupTitle := range groupTitles {
			groupTitle = sanitizeTxtField(groupTitle)
			if groupTitle == "" {
				groupTitle = DefaultGroupName
			}
			addGroup(groupTitle)
			groupEntries[groupTitle] = append(groupEntries[groupTitle], entry{name: name, url: url})
		}
	}

	var b strings.Builder
	for _, group := range groupOrder {
		entries := groupEntries[group]
		if len(entries) == 0 {
			continue
		}
		b.WriteString(group)
		b.WriteString(",#genre#\n")
		for _, e := range entries {
			b.WriteString(e.name)
			b.WriteString(",")
			b.WriteString(e.url)
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func sanitizeTxtField(s string) string {
	s = sanitizeM3UText(s)
	s = strings.ReplaceAll(s, ",", " ")
	return strings.TrimSpace(s)
}
