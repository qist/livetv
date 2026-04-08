package model

type Group struct {
	ID   uint   `gorm:"primary_key"`
	Name string `gorm:"unique_index"`
}
