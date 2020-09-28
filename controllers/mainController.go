package controllers

import  (
	"oms/models"
)

const (
	HttpStatusOk    = "200"
	HttpStatusError = "400"
)

type ResponseGet struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ResponsePost struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

type MainController struct {
	BashController
}

// 首页主机页
func (c *MainController) Get() {
	hosts := models.GetAllHost()
	groups := models.GetAllGroup()
	tags := models.GetAllTag()
	c.Data["Hosts"] = hosts
	c.Data["Groups"] = groups
	c.Data["Tags"] = tags
	c.Layout = "base/layout.html"
	c.TplName = "index.html"
	c.Render()
}

func (c *MainController) GroupPage() {
	groups := models.GetAllGroup()
	tags := models.GetAllTag()
	c.Data["Groups"] = groups
	c.Data["Tags"] = tags
	c.Layout = "base/layout.html"
	c.TplName = "group.html"
	c.Render()
}

func (c *MainController) SshPage() {
	idStr := c.Input().Get("id")
	hosts := models.GetAllHost()
	c.Data["Hosts"] = hosts
	c.Data["HostId"] = idStr
	c.Layout = "base/layout.html"
	c.TplName = "ssh.html"
	c.Render()
}

func (c *MainController) ToolPage() {
	c.Layout = "base/layout.html"
	c.TplName = "tool.html"
	c.Render()
}

func (c *MainController) ShellPage() {
	dType := c.Input().Get("type")
	idStr := c.Input().Get("id")
	c.Data["dType"] = dType
	c.Data["idStr"] = idStr
	c.Layout = "base/layout.html"
	c.TplName = "shell.html"
	c.Render()
}

func (c *MainController) FilePage() {
	dType := c.Input().Get("type")
	idStr := c.Input().Get("id")
	c.Data["dType"] = dType
	c.Data["idStr"] = idStr
	c.Layout = "base/layout.html"
	c.TplName = "file.html"
	c.Render()
}

func (c *MainController) FileBrowsePage() {
	idStr := c.Input().Get("id")
	hosts := models.GetAllHost()
	c.Data["Hosts"] = hosts
	c.Data["HostId"] = idStr
	c.Layout = "base/layout.html"
	c.TplName = "browse.html"
	c.Render()
}
