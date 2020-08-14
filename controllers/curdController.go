package controllers

import (
	"github.com/astaxie/beego"
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
	//data := &Response{HttpStatusOk, "success",
	//	hosts}
	//c.Data["json"] = data
	c.ServeJSON()
}
