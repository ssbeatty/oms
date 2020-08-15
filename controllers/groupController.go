package controllers

import "oms/models"

func (c *GroupController) Get() {
	groups := models.GetAllGroup()
	data := &ResponseGet{HttpStatusOk, "success",
		groups}
	c.Data["json"] = data
	c.ServeJSON()
}
