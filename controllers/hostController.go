package controllers

import (
	"encoding/json"
	"io/ioutil"
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
	var keyText string
	hostname := c.Input().Get("hostname")
	user := c.Input().Get("user")
	addr := c.Input().Get("addr")
	port, _ := c.GetInt("port")
	if port == 0 {
		port = 22
	}
	password := c.Input().Get("password")
	groupId, _ := c.GetInt("group")
	_, fh, err := c.GetFile("keyFile")
	tags := c.Input().Get("tags")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			logger.Logger.Println(err)
		}
		keyText = string(fileBytes)
	}

	err = json.Unmarshal([]byte(tags), &tagJson)
	if err != nil {
		logger.Logger.Println(err)
	}

	models.InsertHost(hostname, user, addr, port, password, groupId, tagJson, keyText)
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *HostController) Put() {
	var msg = "success"
	var code = HttpStatusOk
	var tagJson []string
	var keyText string
	id, err := c.GetInt("id")
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	hostname := c.Input().Get("hostname")
	user := c.Input().Get("user")
	addr := c.Input().Get("addr")
	port, _ := c.GetInt("port")
	password := c.Input().Get("password")
	groupId, _ := c.GetInt("group")
	_, fh, err := c.GetFile("keyFile")

	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			logger.Logger.Println(err)
		}
		keyText = string(fileBytes)
	}
	tags := c.Input().Get("tags")

	err = json.Unmarshal([]byte(tags), &tagJson)
	if err != nil {
		msg = "can't get param tags"
		code = HttpStatusError
		logger.Logger.Println(err)
	}

	models.UpdateHost(id, hostname, user, addr, port, password, groupId, tagJson, keyText)
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
