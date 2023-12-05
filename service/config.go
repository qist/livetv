package service

import (
	"github.com/jinzhu/gorm"
	"github.com/qist/livetv/global"
	"github.com/qist/livetv/model"
)

func GetConfig(key string) (string, error) {
	if confValue, ok := global.ConfigCache.Load(key); ok {
		return confValue.(string), nil
	} else {
		var value model.Config
		err := global.DB.Where("name = ?", key).First(&value).Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return "", global.ErrConfigNotFound
			} else {
				return "", err
			}
		} else {
			global.ConfigCache.Store(key, value.Data)
			return value.Data, nil
		}
	}
}

func SetConfig(key, value string) error {
	data := model.Config{Name: key, Data: value}
	err := global.DB.Save(&data).Error
	if err == nil {
		global.ConfigCache.Store(key, value)
	}
	return err
}
