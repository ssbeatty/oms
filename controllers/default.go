package controllers

import (
	"github.com/astaxie/beego"
	"oms/models"
)

type MainController struct {
	beego.Controller
}

// 首页主机页
func (c *MainController) Get() {
	host := new(models.Host)
	var hosts []*models.Host
	tag := new(models.Tag)
	var tags []*models.Tag
	group := new(models.Group)
	var groups []*models.Group

	_, err := o.QueryTable(host).All(&hosts)
	if err != nil {
		Logger.Println(err)
	}

	// 获取tags
	for i := 0; i < len(hosts); i++ {
		_, err := o.QueryTable(group).Filter("Id", hosts[i].Id).All(&groups)
		_, err = o.QueryTable(tag).Filter("Hosts__Host__Id", hosts[i].Id).All(&tags)
		if err != nil {
			Logger.Println(err)
		}
		hosts[i].Tags = tags
		if len(groups) != 0 {
			hosts[i].Group = groups[0]
		}
		tags = nil
		groups = nil
	}
	c.Data["Hosts"] = hosts
	c.TplName = "index.html"
	c.Render()
}
