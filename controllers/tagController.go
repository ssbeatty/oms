package controllers

import "oms/models"

func (c *TagController) Get() {
	tags := models.GetAllTag()
	data := &ResponseGet{HttpStatusOk, "success",
		tags}
	c.Data["json"] = data
	c.ServeJSON()
}
