package models

import (
	"github.com/astaxie/beego/orm"
	"oms/logger"
	"strconv"
)

// Model Struct
type Host struct {
	Id       int
	Name     string `orm:"size(100)"`
	Addr     string `orm:"null"`
	Port     int    `orm:"default(22)"`
	PassWord string `orm:"null" json:"-"`
	KeyFile  string `orm:"null" json:"-"`

	Group *Group `orm:"rel(fk);null;on_delete(set_null)"`
	Tags  []*Tag `orm:"rel(m2m);null;rel_table(Tag)"`
}

func DeleteHostById(id int) bool {
	o := orm.NewOrm()
	host := Host{Id: id}
	_, err := o.Delete(&host)
	if err != nil {
		logger.Logger.Println(err)
		return false
	}
	return true
}

func InsertHost(hostname string, addr string, port int, password string, groupId int, tags []string, filePath string) *Host {
	var o = orm.NewOrm()
	group := Group{Id: groupId}
	err := o.Read(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	host := Host{
		Name:     hostname,
		Addr:     addr,
		Port:     port,
		PassWord: password,
		Group:    &group,
		KeyFile:  filePath,
	}
	_, err = o.Insert(&host)
	if err != nil {
		logger.Logger.Println(err)
	}
	m2m := o.QueryM2M(&host, "Tags")
	for _, tagIdStr := range tags {
		tagId, _ := strconv.Atoi(tagIdStr)
		tag := Tag{Id: tagId}
		err := o.Read(&tag)
		_, err = m2m.Add(tag)
		if err != nil {
			logger.Logger.Println(err)
		}
	}
	return &host
}

func GetAllHost() []Host {
	var o = orm.NewOrm()
	host := new(Host)
	var hosts []Host
	_, err := o.QueryTable(host).RelatedSel().All(&hosts)
	if err != nil {
		logger.Logger.Println(err)
	}
	// 获取tags
	for i := 0; i < len(hosts); i++ {
		o.LoadRelated(&hosts[i], "Tags")
	}
	return hosts
}
