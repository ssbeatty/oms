package models

import (
	"github.com/astaxie/beego/orm"
	"oms/logger"
)

func ParseHostList(pType string, id int) []*Host {
	var hosts []*Host
	var o = orm.NewOrm()
	if pType == "host" {
		host := Host{Id: id}
		err := o.Read(&host)
		if err != nil {
			logger.Logger.Println(err)
		}
		hosts = append(hosts, &host)
	} else if pType == "tag" {
		host := new(Host)
		_, err := o.QueryTable(host).Filter("Tags__Tag__Id", id).All(&hosts)
		if err != nil {
			logger.Logger.Println(err)
		}
	} else {
		group := Group{Id: id}
		err := o.Read(&group)
		if err != nil {
			logger.Logger.Println(err)
		}
		if group.Mode == 0 {
			host := new(Host)
			_, err = o.QueryTable(host).Filter("Group__Id", id).All(&hosts)
			if err != nil {
				logger.Logger.Println(err)
			}
		} else {
			// TODO mode params
			// something like 192.168.* or -L'a,b,v' -E re
		}

	}
	return hosts
}
