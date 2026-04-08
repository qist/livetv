package model

type Channel struct {
	ID        uint   `gorm:"primary_key"`
	CustomID  string `gorm:"unique_index"`
	Name      string
	URL       string
	Proxy     bool
	GroupName string `gorm:"column:group_name;index"`
	GroupID   uint   `gorm:"column:group_id;index"`
	Group     Group  `gorm:"foreignkey:GroupID"`
}
