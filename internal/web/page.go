package web

import (
	"github.com/gin-gonic/gin"
	wsl "github.com/gorilla/websocket"
	"net/http"
	"oms/internal/models"
	"strconv"
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

var upGrader = wsl.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Service) GetWebsocketIndex(c *gin.Context) {
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Errorf("upgrade websocket failed, err: %v", err)
	}
	ws := NewWSConnect(wsConn, s).InitHandlers()
	ws.Serve()
}

func (s *Service) GetWebsocketSsh(c *gin.Context) {
	idStr := c.Param("id")
	// get pty windows size
	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	id, _ := strconv.Atoi(idStr)
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Errorf("upgrade websocket failed, err: %v", err)
	}
	defer wsConn.Close()

	host, _ := models.GetHostById(id)
	client, err := s.sshManager.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		s.logger.Errorf("transport new client failed, err: %v", err)
	}

	ssConn, err := NewSshConn(cols, rows, client)
	if err != nil {
		s.logger.Errorf("new ssh connect failed, err: %v", err)
	}
	ws := NewWSConnect(wsConn, nil)
	ssConn.SetOutput(ws)
	defer ssConn.Close()
	defer ws.Close()

	quitChan := make(chan bool, 3)
	go ssConn.ReceiveWsMsg(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	s.logger.Info("websocket ssh finished")
}
