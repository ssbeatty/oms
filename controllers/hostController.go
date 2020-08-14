package controllers

import (
	"github.com/astaxie/beego"
	"oms/logger"
	"oms/models"
	"strconv"
)

// host
type HostController struct {
	beego.Controller
}

func (c *HostController) Get() {
	hosts := models.GetAllHost()
	data := &ResponseGet{HttpStatusOk, "success",
		hosts}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *HostController) Post() {

	//data := &ResponseGet{HttpStatusOk, "success",
	//	hosts}
	//c.Data["json"] = data
	c.ServeJSON()
}

func (c *HostController) Delete() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	result := models.DeleteHostById(id)
	if !result {
		msg = "Can't delete object"
		code = HttpStatusError
	}
	logger.Logger.Println(msg)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}
