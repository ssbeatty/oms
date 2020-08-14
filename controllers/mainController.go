package controllers

import (
	"github.com/astaxie/beego"
	"oms/models"
)

type MainController struct {
	beego.Controller
}

type GroupController struct {
	beego.Controller
}

// 首页主机页
func (c *MainController) Get() {
	hosts := models.GetAllHost()
	c.Data["Hosts"] = hosts
	c.TplName = "index.html"
	c.Render()
}

func (c *GroupController) Get() {
	c.TplName = "group.html"
	c.Render()
}
