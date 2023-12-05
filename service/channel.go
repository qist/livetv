package service

import (
	"github.com/qist/livetv/global"
	"github.com/qist/livetv/model"
)

func GetAllChannel() (channels []model.Channel, err error) {
	err = global.DB.Find(&channels).Error
	return
}

func SaveChannel(channel model.Channel) error {
	return global.DB.Save(&channel).Error
}

func DeleteChannel(id uint) error {
	return global.DB.Delete(model.Channel{}, "id = ?", id).Error
}

func GetChannel(channelNumber uint) (channel model.Channel, err error) {
	err = global.DB.Where("id = ?", channelNumber).First(&channel, channelNumber).Error
	return
}
