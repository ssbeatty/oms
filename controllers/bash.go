package controllers

import "github.com/astaxie/beego"

type BashController struct {
	beego.Controller
}

func (c *BashController) Prepare() {
	session := c.GetSession("user")
	if session == nil {
		c.Ctx.Redirect(302, "/login")
	}
}
