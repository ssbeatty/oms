package controllers

import (
	"github.com/astaxie/beego"
	"oms/models"
	"oms/utils"
)

type LoginController struct {
	beego.Controller
}

func (c *LoginController) Get(){
	username := c.GetSession("user")
	if username != nil {
		c.Ctx.Redirect(302,"/")
	}
	c.TplName = "login.html"
	c.Render()
}

func (c *LoginController) Post(){
	var msg = "verify password failed"
	var code = HttpStatusError
	input_phone := c.Input().Get("phone")
	input_password := c.Input().Get("password")
	user := models.GetUserPhone(input_phone)
	// verify password
	if utils.NewMd5(input_password) == user.UserPassword {
		msg = "success"
		code = HttpStatusOk
		c.SetSession("user", user.Name)
		c.SetSession("user_model",*user)
	}
	data := &ResponseGet{code, msg, user}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *LoginController) GetUser(){
	c.Data["user_model"] = c.GetSession("user_model").(models.User)
	c.Layout = "base/layout.html"
	c.TplName = "user_info.html"
	c.Render()
}

func (c *LoginController) EditUser(){
	msg := "user update failed"
	code := HttpStatusError
	user := models.User{}
	update_map := map[string]interface{}{}
	get_name := c.Input().Get("Name")
	if len(get_name)!= 0 {
		user.Name = get_name
		update_map["name"] = get_name
	}
	get_nickname := c.Input().Get("NickName")
	if len(get_nickname)!= 0 {
		user.NickName = get_nickname
		update_map["nick_name"] = get_nickname
	}
	get_phone := c.Input().Get("Phone")
	if len(get_phone)!= 0 && len(get_phone) == 11 {
		user.Phone = get_phone
		update_map["phone"] = get_phone
	}
	get_emil := c.Input().Get("Emil")
	if len(get_emil)!= 0 {
		user.Emil = get_emil
		update_map["emil"] = get_emil
	}
	get_pass1 := c.Input().Get("UserPassword")
	if len(get_pass1)!= 0 {
		user.UserPassword = utils.NewMd5(get_pass1)
		update_map["user_password"] = utils.NewMd5(get_pass1)
	}
	err := models.UpdateUser(update_map)
	if err != nil {
		c.SetSession("user", user.Name)
		c.SetSession("user_model",user)
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

func (c *LoginController) LogOut(){
	c.DelSession("user")
	c.DelSession("user_model")
	c.Ctx.Redirect(302,"/login")
}
