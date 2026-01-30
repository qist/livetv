package model

type Channel struct {
	ID       uint   `gorm:"primary_key"`
	CustomID string `gorm:"unique_index"`
	Name     string
	URL      string
	Proxy    bool
}
