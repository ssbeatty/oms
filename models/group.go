package models

import (
	"github.com/astaxie/beego/orm"
	"oms/logger"
)

type Group struct {
	Id     int
	Name   string  `orm:"size(100)"`
	Mode   int     `orm:"default(0)" json:"-"` //0.Default mode & use Host, 1.Use other func
	Host   []*Host `orm:"reverse(many);null" json:"-"`
	Params string  `orm:"null" json:"-"`
}

func GetAllGroup() []Group {
	var o = orm.NewOrm()
	group := new(Group)
	var groups []Group
	_, err := o.QueryTable(group).RelatedSel().All(&groups)
	if err != nil {
		logger.Logger.Println(err)
	}
	return groups
}
