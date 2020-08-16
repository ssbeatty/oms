package models

import (
	"github.com/astaxie/beego/orm"
	"oms/logger"
)

type Group struct {
	Id     int
	Name   string  `orm:"size(100)"`
	Mode   int     `orm:"default(0)"` //0.Default mode & use Host, 1.Use other func
	Host   []*Host `orm:"reverse(many);null" json:"-"`
	Params string  `orm:"null"`
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

func GetGroupById(id int) *Group {
	var o = orm.NewOrm()
	group := Group{Id: id}
	err := o.Read(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	return &group
}

func InsertGroup(name string, params string, mode int) *Group {
	var o = orm.NewOrm()
	group := Group{
		Name:   name,
		Params: params,
		Mode:   mode,
	}
	_, err := o.Insert(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	return &group
}

func UpdateGroup(id int, name string, params string, mode int) *Group {
	var o = orm.NewOrm()
	group := Group{Id: id}
	err := o.Read(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	if name != "" {
		group.Name = name
	}
	if params != "" {
		group.Params = params
	}
	group.Mode = mode
	_, err = o.Update(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	return &group
}

func DeleteGroupById(id int) bool {
	o := orm.NewOrm()
	tag := Group{Id: id}
	_, err := o.Delete(&tag)
	if err != nil {
		logger.Logger.Println(err)
		return false
	}
	return true
}
