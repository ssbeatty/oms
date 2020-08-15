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

func GetAllTag() []Tag {
	var o = orm.NewOrm()
	tag := new(Tag)
	var tags []Tag
	_, err := o.QueryTable(tag).RelatedSel().All(&tags)
	if err != nil {
		logger.Logger.Println(err)
	}
	return tags
}
