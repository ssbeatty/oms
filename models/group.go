package models

import (
	"log"
)

type Group struct {
	Id     int
	Name   string `gorm:"size:256;not null"`
	Mode   int    `gorm:"default:0;not null"` //0.主机模式, 1.其他匹配模式主机不生效
	Host   []Host `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Params string
}

func GetAllGroup() []*Group {
	var groups []*Group
	result := db.Find(&groups)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return groups
}

func GetGroupById(id int) *Group {
	group := Group{}
	result := db.Where("id = ?", id).First(&group)

	if result.Error != nil {
		log.Println(result.Error)
	}
	return &group
}

func ExistedGroup(name string) bool {
	var groups []*Group
	result := db.Where("name = ?", name).Find(&groups)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}
	if len(groups) == 0 {
		return false
	}
	return true
}

func InsertGroup(name string, params string, mode int) *Group {
	group := Group{
		Name:   name,
		Params: params,
		Mode:   mode,
	}
	result := db.Create(&group)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &group
}

func UpdateGroup(id int, name string, params string, mode int) *Group {
	group := Group{}
	result := db.Where("id = ?", id).First(&group)
	if result.Error != nil {
		log.Println(result.Error)
	}
	if name != "" {
		group.Name = name
	}
	if params != "" {
		group.Params = params
	}
	group.Mode = mode
	result = db.Save(&group)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &group
}

func DeleteGroupById(id int) bool {
	result := db.Delete(&Group{}, id)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}
	return true
}
