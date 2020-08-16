package controllers

import (
	"encoding/json"
	"oms/logger"
	"oms/models"
	"strconv"
)

func (c *HostController) Get() {
	hosts := models.GetAllHost()
	data := &ResponseGet{HttpStatusOk, "success",
		hosts}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *HostController) Post() {
	var msg = "success"
	var code = HttpStatusOk
	var tagJson []string
	hostname := c.Input().Get("hostname")
	addr := c.Input().Get("addr")
	port, _ := c.GetInt("port")
	if port == 0 {
		port = 22
	}
	password := c.Input().Get("password")
	groupId, _ := c.GetInt("group")
	file, _, _ := c.GetFile("keyFile")
	tags := c.Input().Get("tags")

	err := json.Unmarshal([]byte(tags), &tagJson)
	if err != nil {
		logger.Logger.Println(err)
	}
	filePath := getFileName()
	if file != nil {
		defer file.Close()
	}
	err = c.SaveToFile("keyFile", filePath)
	if err != nil {
		filePath = ""
		logger.Logger.Println(err)
	}
	models.InsertHost(hostname, addr, port, password, groupId, tagJson, filePath)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *HostController) Put() {
	var msg = "success"
	var code = HttpStatusOk
	var tagJson []string
	id, err := c.GetInt("id")
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	hostname := c.Input().Get("hostname")
	addr := c.Input().Get("addr")
	port, _ := c.GetInt("port")
	password := c.Input().Get("password")
	groupId, _ := c.GetInt("group")
	file, _, _ := c.GetFile("keyFile")
	tags := c.Input().Get("tags")

	err = json.Unmarshal([]byte(tags), &tagJson)
	if err != nil {
		msg = "can't get param tags"
		code = HttpStatusError
		logger.Logger.Println(err)
	}
	filePath := getFileName()
	if file != nil {
		defer file.Close()
	}
	err = c.SaveToFile("keyFile", filePath)
	if err != nil {
		filePath = ""
		logger.Logger.Println(err)
	}
	models.UpdateHost(id, hostname, addr, port, password, groupId, tagJson, filePath)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
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

func (c *HostController) GetOneHost() {
	var msg = "success"
	var code = HttpStatusOk
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.Atoi(idStr)
	hosts := models.GetHostById(id)
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	data := &ResponseGet{code, msg,
		hosts}
	c.Data["json"] = data
	c.ServeJSON()
}
