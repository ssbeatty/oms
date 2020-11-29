package page

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"oms/logger"
	"oms/models"
	"oms/ssh"
	"strconv"
)

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GetIndexPage(c *gin.Context) {
	hosts := models.GetAllHost()
	groups := models.GetAllGroup()
	tags := models.GetAllTag()

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Hosts":  hosts,
		"Groups": groups,
		"Tags":   tags,
	})
}

func GetGroupPage(c *gin.Context) {
	groups := models.GetAllGroup()
	tags := models.GetAllTag()

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
	hosts := models.GetAllHost()

	c.HTML(http.StatusOK, "browse.html", gin.H{
		"HostId": HostId,
		"Hosts":  hosts,
	})
}

func GetSshPage(c *gin.Context) {
	HostId := c.Query("id")
	hosts := models.GetAllHost()

	c.HTML(http.StatusOK, "ssh.html", gin.H{
		"HostId": HostId,
		"Hosts":  hosts,
	})
}

func GetWebsocket(c *gin.Context) {
	idStr := c.Param("id")
	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	id, _ := strconv.Atoi(idStr)
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Logger.Println(err)
	}
	defer wsConn.Close()
	host := models.GetHostById(id)
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		logger.Logger.Println(err)
	}
	ssConn, err := ssh.NewSshConn(cols, rows, client.SSHClient)
	defer ssConn.Close()
	quitChan := make(chan bool, 3)
	var logBuff = new(bytes.Buffer)
	go ssConn.ReceiveWsMsg(wsConn, logBuff, quitChan)
	go ssConn.SendComboOutput(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	logger.Logger.Println("websocket finished")
}
