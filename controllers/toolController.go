package controllers

import (
	"oms/logger"
	"oms/models"
	"strconv"
)

func (c *ToolController) RunCmd() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	pType := c.Input().Get("type")
	hosts := models.ParseHostList(pType, id)
	// do cmd

	data := &ResponseGet{code, msg,
		hosts}
	c.Data["json"] = data
	c.ServeJSON()
}
