package controllers

import (
	"github.com/astaxie/beego"
	"oms/models"
)

type MainController struct {
	beego.Controller
}

// resources
type GroupController struct {
	beego.Controller
}

type HostController struct {
	beego.Controller
}

type TagController struct {
	beego.Controller
}

// 首页主机页
func (c *MainController) Get() {
	hosts := models.GetAllHost()
	groups := models.GetAllGroup()
	tags := models.GetAllTag()
	c.Data["Hosts"] = hosts
	c.Data["Groups"] = groups
	c.Data["Tags"] = tags
	c.Layout = "base/layout.html"
	c.TplName = "index.html"
	c.Render()
}

func (c *MainController) GroupPage() {
	c.Layout = "base/layout.html"
	c.TplName = "group.html"
	c.Render()
}
