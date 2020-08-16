package controllers

import (
	"oms/logger"
	"oms/models"
	"strconv"
)

func (c *TagController) Get() {
	tags := models.GetAllTag()
	data := &ResponseGet{HttpStatusOk, "success",
		tags}
	c.Data["json"] = data
	c.ServeJSON()
}
func (c *TagController) GetOneTag() {
	var msg = "success"
	var code = HttpStatusOk
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.Atoi(idStr)
	tag := models.GetTagById(id)
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	data := &ResponseGet{code, msg,
		tag}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *TagController) Post() {
	var msg = "success"
	var code = HttpStatusOk
	name := c.Input().Get("name")

	models.InsertTag(name)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *TagController) Put() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := c.GetInt("id")
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	name := c.Input().Get("name")

	models.UpdateTag(id, name)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *TagController) Delete() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	result := models.DeleteTagById(id)
	if !result {
		msg = "Can't delete object"
		code = HttpStatusError
	}
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}
