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
	qs := o.QueryTable(user)

	fmt.Println(qs)
	//data := &Response{"100", "获取成功",
	//	userList}
	//c.Data["json"] = data
	//c.ServeJSON()
}
