package global

import (
	"log"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/qist/livetv/model"
)

var DB *gorm.DB

func InitDB(filepath string) (err error) {
	DB, err = gorm.Open("sqlite3", filepath)
	if err != nil {
		return err
	}
	err = DB.AutoMigrate(&model.Config{}, &model.Group{}, &model.Channel{}).Error
	if err != nil {
		return err
	}
	if err := migrateChannelGroups(); err != nil {
		return err
	}
	for key, valueDefault := range defaultConfigValue {
		var valueInDB model.Config
		err = DB.Where("name = ?", key).First(&valueInDB).Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				// Persist defaults so config exists across restarts.
				if createErr := DB.Create(&model.Config{Name: key, Data: valueDefault}).Error; createErr != nil {
					log.Printf("init: unable to persist default config %q: %v", key, createErr)
				}
				ConfigCache.Store(key, valueDefault)
			} else {
				return err
			}
		} else {
			ConfigCache.Store(key, valueInDB.Data)
		}
	}
	return nil
}

func migrateChannelGroups() error {
	const defaultGroupName = "youtube"

	var channels []model.Channel
	if err := DB.Select("id, group_id, group_name").Find(&channels).Error; err != nil {
		return err
	}
	for _, ch := range channels {
		groupName := strings.TrimSpace(ch.GroupName)
		if groupName == "" {
			groupName = defaultGroupName
		}
		var group model.Group
		findErr := DB.Where("name = ?", groupName).First(&group).Error
		if findErr != nil {
			if gorm.IsRecordNotFoundError(findErr) {
				group = model.Group{Name: groupName}
				if err := DB.Create(&group).Error; err != nil {
					return err
				}
			} else {
				return findErr
			}
		}
		if ch.GroupID == 0 || ch.GroupID != group.ID || ch.GroupName != group.Name {
			if err := DB.Model(&model.Channel{}).Where("id = ?", ch.ID).Updates(map[string]interface{}{
				"group_id":   group.ID,
				"group_name": group.Name,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
