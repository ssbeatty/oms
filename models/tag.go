package models

import (
	"log"
)

type Tag struct {
	Id    int
	Name  string `gorm:"size:100"`
	Hosts []Host `gorm:"many2many:host_tag;"`
}

func GetAllTag() []*Tag {
	var tags []*Tag
	result := db.Find(&tags)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return tags
}

func GetTagById(id int) *Tag {
	tag := Tag{}
	result := db.Where("id = ?", id).First(&tag)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &tag
}

func ExistedTag(name string) bool {
	var tags []*Tag
	result := db.Where("name = ?", name).Find(&tags)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}
	if len(tags) == 0 {
		return false
	}
	return true
}

func InsertTag(name string) *Tag {
	tag := Tag{
		Name: name,
	}
	result := db.Create(&tag)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &tag
}

func UpdateTag(id int, name string) *Tag {
	tag := Tag{Id: id}
	result := db.Where("id = ?", id).First(&tag)
	if result.Error != nil {
		log.Println(result.Error)
	}
	if name != "" {
		tag.Name = name
	}
	result = db.Save(&tag)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &tag
}

func DeleteTagById(id int) bool {
	tag := Tag{}
	result := db.Where("id = ?", id).First(&tag)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}
	if err := db.Model(&tag).Association("Hosts").Clear(); err != nil {
		log.Println("clear association error for tags and host.")
	}
	result = db.Delete(&tag)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}
	return true
}
