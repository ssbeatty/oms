package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"oms/models"
)

type HostController struct {
	beego.Controller
}

func (c *HostController) Get() {

	o := orm.NewOrm()
	user := new(models.Host)
	var hosts []models.Host
	_, err := o.QueryTable(user).All(&hosts)
	if err != nil {
		panic(err)
	}
	for _, host := range hosts {
		fmt.Println(host)
	}
	data := &Response{HttpStatusOk, "获取成功",
		hosts}
	c.Data["json"] = data
	c.ServeJSON()
}
