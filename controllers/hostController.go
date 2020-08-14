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
	//user := new(models.Host)
	//var hosts []models.Host
	//_, err := o.QueryTable(user).All(&hosts)
	//if err != nil {
	//	panic(err)
	//}
	//for _, host := range hosts {
	//	Logger.Println(host)
	//}
	//data := &ResponseGet{HttpStatusOk, "success",
	//	hosts}
	//c.Data["json"] = data
	c.ServeJSON()
}

func (c *HostController) Post() {
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
