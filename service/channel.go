package service

import (
	"strconv"
	"strings"

	"github.com/qist/livetv/global"
	"github.com/qist/livetv/model"
)

const DefaultGroupName = "youtube"

func GetAllChannel() (channels []model.Channel, err error) {
	err = global.DB.Preload("Group").Find(&channels).Error
	if err != nil {
		return
	}
	for i := range channels {
		if channels[i].GroupID != 0 && strings.TrimSpace(channels[i].Group.Name) != "" {
			channels[i].GroupName = channels[i].Group.Name
		}
	}
	return
}

func SaveChannel(channel model.Channel) error {
	channel.URL = normalizeYoutubeURL(channel.URL)
	channel.GroupName = strings.TrimSpace(channel.GroupName)
	if channel.GroupName == "" {
		channel.GroupName = DefaultGroupName
	}
	group, err := EnsureGroupByName(channel.GroupName)
	if err != nil {
		return err
	}
	channel.GroupID = group.ID
	channel.GroupName = group.Name
	if channel.ID == 0 {
		err := global.DB.Create(&channel).Error
		if err != nil {
			return err
		}
		if channel.CustomID == "" {
			autoID := strconv.Itoa(int(channel.ID))
			return global.DB.Model(&channel).Update("custom_id", autoID).Error
		}
		return nil
	}
	return global.DB.Save(&channel).Error
}

func DeleteChannel(id uint) error {
	return global.DB.Delete(model.Channel{}, "id = ?", id).Error
}

func GetChannel(channelIdentifier interface{}) (channel model.Channel, err error) {
	// Try custom ID first
	err = global.DB.Preload("Group").Where("custom_id = ?", channelIdentifier).First(&channel).Error
	if err == nil {
		if channel.GroupID != 0 && strings.TrimSpace(channel.Group.Name) != "" {
			channel.GroupName = channel.Group.Name
		}
		return
	}
	// Fall back to numeric ID
	err = global.DB.Preload("Group").Where("id = ?", channelIdentifier).First(&channel).Error
	if err == nil {
		if channel.GroupID != 0 && strings.TrimSpace(channel.Group.Name) != "" {
			channel.GroupName = channel.Group.Name
		}
	}
	return
}
