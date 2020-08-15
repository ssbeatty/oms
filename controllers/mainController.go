package controllers

import (
	"github.com/astaxie/beego"
	"oms/models"
)

type MainController struct {
	beego.Controller
}

type TestController struct {
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

//path
type GroupPathController struct {
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
	c.TplName = "index.html"
	c.Render()
}

func (c *GroupPathController) Get() {
	c.TplName = "group.html"
	c.Render()
}

func (c *TestController) Get() {
	models.TestFunc()
}
