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
	PassWord string `orm:"null"`
	KeyFile  string `orm:"null"`

	Group *Group `orm:"rel(fk);null;on_delete(set_null)"`
	Tags  []*Tag `orm:"rel(m2m);null;rel_table(Tag)"`
}

func GetHostById(id int) *Host {
	var o = orm.NewOrm()
	host := Host{Id: id}
	err := o.Read(&host)
	o.LoadRelated(&host, "Tags")
	o.LoadRelated(&host, "Group")
	if err != nil {
		logger.Logger.Println(err)
	}
	return &host
}

func DeleteHostById(id int) bool {
	o := orm.NewOrm()
	host := Host{Id: id}
	m2m := o.QueryM2M(&host, "Tags")
	_, err := m2m.Clear()
	if err != nil {
		logger.Logger.Println(err)
		return false
	}
	_, err = o.Delete(&host)
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

func UpdateHost(id int, hostname string, addr string, port int, password string, groupId int, tags []string, filePath string) *Host {
	var o = orm.NewOrm()
	group := Group{Id: groupId}
	err := o.Read(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	host := Host{Id: id}
	err = o.Read(&host)
	if err != nil {
		logger.Logger.Println(err)
	}
	if hostname != "" {
		host.Name = hostname
	}
	if port != 0 {
		host.Port = port
	}
	if addr != "" {
		host.Addr = addr
	}
	if password != "" {
		host.PassWord = password
	}
	if filePath != "" {
		host.KeyFile = filePath
	}
	if groupId != 0 {
		host.Group = &group
	}
	_, err = o.Update(&host)
	if err != nil {
		logger.Logger.Println(err)
	}
	m2m := o.QueryM2M(&host, "Tags")
	_, err = m2m.Clear()
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
