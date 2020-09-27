package controllers

import (
	"oms/logger"
	"oms/models"
	"strconv"
)

type GroupController struct {
	BashController
}

func (c *GroupController) Get() {
	groups := models.GetAllGroup()
	data := &ResponseGet{HttpStatusOk, "success",
		groups}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *GroupController) GetOneGroup() {
	var msg = "success"
	var code = HttpStatusOk
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.Atoi(idStr)
	group := models.GetGroupById(id)
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	data := &ResponseGet{code, msg,
		group}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *GroupController) Post() {
	var msg = "success"
	var code = HttpStatusOk
	name := c.Input().Get("name")
	params := c.Input().Get("params")
	mode, err := c.GetInt("mode")
	if err != nil {
		msg = "Error params mode"
		code = HttpStatusError
	}
	models.InsertGroup(name, params, mode)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *GroupController) Put() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := c.GetInt("id")
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	name := c.Input().Get("name")
	params := c.Input().Get("params")
	mode, err := c.GetInt("mode")
	if err != nil {
		msg = "Error params mode"
		code = HttpStatusError
	}

	models.UpdateGroup(id, name, params, mode)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *GroupController) Delete() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	result := models.DeleteGroupById(id)
	if !result {
		msg = "Can't delete object"
		code = HttpStatusError
	}
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}
