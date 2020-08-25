package controllers

import (
	"bytes"
	"github.com/astaxie/beego"
	"io/ioutil"
	"net/http"
	"oms/logger"
	"oms/models"
	"strconv"
	"time"
)

type ToolController struct {
	beego.Controller
}

func (c *ToolController) RunCmd() {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	pType := c.Input().Get("type")
	cmd := c.Input().Get("cmd")
	hosts := models.ParseHostList(pType, id)
	// do cmd
	results := models.RunCmd(hosts, cmd)
	data := &ResponseGet{code, msg,
		results}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *ToolController) FileUpload() {
	var msg = "success"
	var code = HttpStatusOk
	var remoteFile string
	id, err := strconv.Atoi(c.Input().Get("id"))
	files, _ := c.GetFiles("files")
	remote := c.GetString("remote")
	if remote[len(remote)-1] == '/' {
		remoteFile = remote
	} else {
		remoteFile = remote + "/"
	}
	if err != nil {
		logger.Logger.Println(err)
	}
	pType := c.Input().Get("type")
	hosts := models.ParseHostList(pType, id)

	results := models.UploadFile(hosts, files, remoteFile)
	data := &ResponseGet{code, msg,
		results}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *ToolController) GetPathInfo() {
	var msg = "success"
	var code = HttpStatusOk
	path := c.Input().Get("path")
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		logger.Logger.Println(err)
		code = HttpStatusError
		msg = err.Error()
	}
	results := models.GetPathInfo(id, path)
	data := &ResponseGet{code, msg,
		results}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *ToolController) DownLoadFile() {
	path := c.Input().Get("path")
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	file := models.DownloadFile(id, path)
	if file != nil {
		fh, err := file.Stat()
		if err != nil {
			logger.Logger.Println(err)
			http.ServeContent(c.Ctx.Output.Context.ResponseWriter, c.Ctx.Output.Context.Request, "download", time.Now(), file)
		} else {
			http.ServeContent(c.Ctx.Output.Context.ResponseWriter, c.Ctx.Output.Context.Request, fh.Name(), fh.ModTime(), file)
		}
	} else {
		data := &ResponsePost{HttpStatusError, "download file error"}
		c.Data["json"] = data
		c.ServeJSON()
	}
}

func (c *ToolController) DeleteFile() {
	var data *ResponsePost
	path := c.Input().Get("path")
	id, err := strconv.Atoi(c.Input().Get("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	err = models.DeleteFileOrDir(id, path)
	if err != nil {
		data = &ResponsePost{HttpStatusError, "remove file error"}
	} else {
		data = &ResponsePost{HttpStatusOk, "success"}
	}
	c.Data["json"] = data
	c.ServeJSON()
}

func (c *ToolController) ExportData() {
	marshal, err := models.ExportDbData()
	if err != nil {
		logger.Logger.Println(err)
	}
	file := bytes.NewReader(marshal)
	c.Ctx.ResponseWriter.Header().Set("content-type", "application/json")
	http.ServeContent(c.Ctx.ResponseWriter, c.Ctx.Request, "export", time.Now(), file)
}

func (c *ToolController) ImportData() {
	var msg = "success"
	var code = HttpStatusOk
	_, fh, err := c.GetFile("dataFile")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			logger.Logger.Println(err)
			msg = err.Error()
			code = HttpStatusError
		}
		err = models.ImportDbData(fileBytes)
		if err != nil {
			logger.Logger.Println(err)
			msg = err.Error()
			code = HttpStatusError
		}
	} else {
		msg = err.Error()
		code = HttpStatusError
	}
	data := &ResponsePost{code, msg}
	c.Data["json"] = data
	c.ServeJSON()
}
