package controllers

import (
	"github.com/astaxie/beego"
	"oms/models"
	"oms/utils"
)

type LoginController struct {
	beego.Controller
}

func (c *LoginController) Get() {
	username := c.GetSession("user")
	if username != nil {
		c.Ctx.Redirect(302, "/")
	}
	c.TplName = "login.html"
	c.Render()
}

func (c *LoginController) Post() {
	var msg = "verify password failed"
	var code = HttpStatusError
	inputPhone := c.Input().Get("phone")
	inputPassword := c.Input().Get("password")
	user := models.GetUserPhone(inputPhone)
	// verify password
	if utils.NewMd5(inputPassword) == user.UserPassword {
		msg = "success"
		code = HttpStatusOk
		c.SetSession("user", user.Name)
		c.SetSession("user_model", *user)
	}
	data := &ResponseGet{code, msg, user}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *LoginController) GetUser() {
	c.Data["user_model"] = c.GetSession("user_model").(models.User)
	c.Layout = "base/layout.html"
	c.TplName = "user_info.html"
	c.Render()
}

func (c *LoginController) EditUser() {
	msg := "user update failed"
	code := HttpStatusError
	user := models.User{}
	updateMap := map[string]interface{}{}
	getName := c.Input().Get("Name")
	if len(getName) != 0 {
		user.Name = getName
		updateMap["name"] = getName
	}
	getNickname := c.Input().Get("NickName")
	if len(getNickname) != 0 {
		user.NickName = getNickname
		updateMap["nick_name"] = getNickname
	}
	getPhone := c.Input().Get("Phone")
	if len(getPhone) != 0 && len(getPhone) == 11 {
		user.Phone = getPhone
		updateMap["phone"] = getPhone
	}
	getEmil := c.Input().Get("Emil")
	if len(getEmil) != 0 {
		user.Emil = getEmil
		updateMap["emil"] = getEmil
	}
	getPass1 := c.Input().Get("UserPassword")
	if len(getPass1) != 0 {
		user.UserPassword = utils.NewMd5(getPass1)
		updateMap["user_password"] = utils.NewMd5(getPass1)
	}
	err := models.UpdateUser(updateMap)
	if err != nil {
		c.SetSession("user", user.Name)
		c.SetSession("user_model", user)
		data := &ResponseGet{code, msg, user}
		c.Data["json"] = data
		c.ServeJSON()
	}
	msg = "user update succeed"
	code = HttpStatusOk
	data := &ResponseGet{code, msg, user}
	c.Data["json"] = data
	c.DelSession("user")
	c.DelSession("user_model")
	c.ServeJSON()
}

func (c *LoginController) LogOut() {
	c.DelSession("user")
	c.DelSession("user_model")
	c.Ctx.Redirect(302, "/login")
}
