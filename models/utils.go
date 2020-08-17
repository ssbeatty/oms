package models

import (
	"github.com/astaxie/beego/orm"
	"oms/logger"
	"oms/ssh"
)

type Result struct {
	Status   bool
	HostId   int
	HostName string
	Msg      string
}

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

func RunCmdOne(host *Host, cmd string) ([]byte, error) {
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		return nil, err
	}
	return client.Output(cmd)
}

func RunCmd(hosts []*Host, cmd string) []*Result {
	var results []*Result

	for _, host := range hosts {
		result, err := RunCmdOne(host, cmd)
		if err != nil {
			results = append(results, &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()})
		} else {
			results = append(results, &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(result)})
		}

	}
	return results
}
