package models

import (
	"github.com/astaxie/beego/orm"
	"oms/logger"
)

type Tag struct {
	Id    int
	Name  string  `orm:"size(100)"`
	Hosts []*Host `orm:"reverse(many);null" json:"-"`
}

func GetAllTag() []*Tag {
	var o = orm.NewOrm()
	tag := new(Tag)
	var tags []*Tag
	_, err := o.QueryTable(tag).RelatedSel().All(&tags)
	if err != nil {
		logger.Logger.Println(err)
	}
	return tags
}

func GetTagById(id int) *Tag {
	var o = orm.NewOrm()
	tag := Tag{Id: id}
	err := o.Read(&tag)
	if err != nil {
		logger.Logger.Println(err)
	}
	return &tag
}

func ExistedTag(name string) bool {
	var o = orm.NewOrm()
	tag := new(Tag)
	var tags []*Tag
	_, err := o.QueryTable(tag).Filter("Name", name).All(&tags)
	if err != nil {
		logger.Logger.Println(err)
		return false
	}
	if len(tags) == 0 {
		return false
	}
	return true
}

func InsertTag(name string) *Tag {
	var o = orm.NewOrm()
	tag := Tag{
		Name: name,
	}
	_, err := o.Insert(&tag)
	if err != nil {
		logger.Logger.Println(err)
	}
	return &tag
}

func UpdateTag(id int, name string) *Tag {
	var o = orm.NewOrm()
	tag := Tag{Id: id}
	err := o.Read(&tag)
	if err != nil {
		logger.Logger.Println(err)
	}
	if name != "" {
		tag.Name = name
	}
	_, err = o.Update(&tag)
	if err != nil {
		logger.Logger.Println(err)
	}
	return &tag
}

func DeleteTagById(id int) bool {
	o := orm.NewOrm()
	tag := Tag{Id: id}
	m2m := o.QueryM2M(&tag, "Hosts")
	_, err := m2m.Clear()
	if err != nil {
		logger.Logger.Println(err)
		return false
	}
	_, err = o.Delete(&tag)
	if err != nil {
		logger.Logger.Println(err)
		return false
	}
	return true
}
