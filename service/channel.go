package service

import (
	"strconv"

	"github.com/qist/livetv/global"
	"github.com/qist/livetv/model"
)

func GetAllChannel() (channels []model.Channel, err error) {
	err = global.DB.Find(&channels).Error
	return
}

func SaveChannel(channel model.Channel) error {
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
	err = global.DB.Where("custom_id = ?", channelIdentifier).First(&channel).Error
	if err == nil {
		return
	}
	// Fall back to numeric ID
	err = global.DB.Where("id = ?", channelIdentifier).First(&channel).Error
	return
}
