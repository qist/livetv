package global

import (
	"log"

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
	err = DB.AutoMigrate(&model.Config{}, &model.Channel{}).Error
	if err != nil {
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
