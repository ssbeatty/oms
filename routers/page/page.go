package page

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"oms/models"
)

func GetIndexPage(c *gin.Context) {
	hosts, _ := models.GetAllHost()
	groups, _ := models.GetAllGroup()
	tags, _ := models.GetAllTag()

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Hosts":  hosts,
		"Groups": groups,
		"Tags":   tags,
	})
}

func GetGroupPage(c *gin.Context) {
	groups, _ := models.GetAllGroup()
	tags, _ := models.GetAllTag()

	c.HTML(http.StatusOK, "group.html", gin.H{
		"Groups": groups,
		"Tags":   tags,
	})
}

func GetToolPage(c *gin.Context) {
	c.HTML(http.StatusOK, "tool.html", nil)
}

func GetShellPage(c *gin.Context) {
	dType := c.Query("type")
	idStr := c.Query("id")

	c.HTML(http.StatusOK, "shell.html", gin.H{
		"dType": dType,
		"idStr": idStr,
	})
}

func GetFilePage(c *gin.Context) {
	dType := c.Query("type")
	idStr := c.Query("id")

	c.HTML(http.StatusOK, "file.html", gin.H{
		"dType": dType,
		"idStr": idStr,
	})
}

func GetFileBrowsePage(c *gin.Context) {
	HostId := c.Query("id")
	hosts, _ := models.GetAllHost()

	c.HTML(http.StatusOK, "browse.html", gin.H{
		"HostId": HostId,
		"Hosts":  hosts,
	})
}

func GetSshPage(c *gin.Context) {
	HostId := c.Query("id")
	hosts, _ := models.GetAllHost()

	c.HTML(http.StatusOK, "ssh.html", gin.H{
		"HostId": HostId,
		"Hosts":  hosts,
	})
}
