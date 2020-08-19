package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"oms/logger"
	"regexp"
	"strconv"
	"strings"
)

// Model Struct
type Host struct {
	Id       int
	Name     string `orm:"size(100)"`
	User     string `orm:"null"`
	Addr     string `orm:"null"`
	Port     int    `orm:"default(22)"`
	PassWord string `orm:"null"`
	KeyFile  string `orm:"null"`

	Status bool `orm:"default(false)"`

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

func InsertHost(hostname string, user string, addr string, port int, password string, groupId int, tags []string, filePath string) *Host {
	var o = orm.NewOrm()
	group := Group{Id: groupId}
	err := o.Read(&group)
	if err != nil {
		logger.Logger.Println(err)
	}
	host := Host{
		Name:     hostname,
		User:     user,
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

func UpdateHost(id int, hostname string, user string, addr string, port int, password string, groupId int, tags []string, filePath string) *Host {
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
	if user != "" {
		host.User = user
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

func GetAllHost() []*Host {
	var o = orm.NewOrm()
	host := new(Host)
	var hosts []*Host
	_, err := o.QueryTable(host).RelatedSel().All(&hosts)
	if err != nil {
		logger.Logger.Println(err)
	}
	// 获取tags
	for i := 0; i < len(hosts); i++ {
		o.LoadRelated(hosts[i], "Tags")
	}
	return hosts
}

func GetHostByGlob(glob string) []*Host {
	var o = orm.NewOrm()
	var hosts []*Host
	glob = strings.Replace(glob, "*", "%", -1)
	sql := fmt.Sprintf("SELECT * FROM host WHERE addr LIKE '%s'", glob)
	_, err := o.Raw(sql).QueryRows(&hosts)
	if err != nil {
		logger.Logger.Println(err)
	}
	return hosts
}

func GetHostByReg(regStr string) []*Host {
	var o = orm.NewOrm()
	host := new(Host)
	var hosts []*Host
	var hostsR []*Host
	num, err := o.QueryTable(host).All(&hosts)
	if err != nil {
		logger.Logger.Println(err)
	}
	for i := 0; i < int(num); i++ {
		match, _ := regexp.MatchString(regStr, hosts[i].Addr)
		if match {
			hostsR = append(hostsR, hosts[i])
		}
	}

	return hostsR
}

func GetHostByAddr(addr string) []*Host {
	var o = orm.NewOrm()
	host := new(Host)
	var hosts []*Host
	_, err := o.QueryTable(host).Filter("Addr", addr).All(&hosts)
	if err != nil {
		logger.Logger.Println(err)
	}
	return hosts
}

func GetHostByKeyFile(KeyFile string) int {
	var o = orm.NewOrm()
	host := new(Host)
	var hosts []*Host
	_, err := o.QueryTable(host).Filter("KeyFile", KeyFile).All(&hosts)
	if err != nil {
		logger.Logger.Println(err)
	}
	return len(hosts)
}
