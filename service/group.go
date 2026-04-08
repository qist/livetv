package service

import (
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/qist/livetv/global"
	"github.com/qist/livetv/model"
)

func GetAllGroups() ([]model.Group, error) {
	var groups []model.Group
	if err := global.DB.Order("name asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func EnsureGroupByName(name string) (model.Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultGroupName
	}
	var group model.Group
	err := global.DB.Where("name = ?", name).First(&group).Error
	if err == nil {
		return group, nil
	}
	if !gorm.IsRecordNotFoundError(err) {
		return group, err
	}
	group = model.Group{Name: name}
	if err := global.DB.Create(&group).Error; err != nil {
		return group, err
	}
	return group, nil
}

func GetAllGroupNames() ([]string, error) {
	if _, err := EnsureGroupByName(DefaultGroupName); err != nil {
		return nil, err
	}
	groups, err := GetAllGroups()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		if strings.TrimSpace(g.Name) == "" {
			continue
		}
		out = append(out, g.Name)
	}
	return out, nil
}
