package controllers

import (
	"github.com/gin-gonic/gin"
	wsl "github.com/gorilla/websocket"
	"net/http"
	"oms/internal/models"
	"oms/internal/web/websocket"
	"strconv"
)

func (s *Service) GetIndexPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
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
		s.Logger.Errorf("upgrade websocket failed, err: %v", err)
	}
	ws := websocket.NewWSConnect(wsConn, s).InitHandlers()
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
		s.Logger.Errorf("upgrade websocket failed, err: %v", err)
	}
	defer wsConn.Close()

	host, err := models.GetHostById(id)
	if err != nil {
		s.Logger.Errorf("can not get host")
		return
	}
	client, err := s.sshManager.NewClient(host)
	if err != nil {
		s.Logger.Errorf("transport new client failed, err: %v", err)
		return
	}

	ssConn, err := websocket.NewSshConn(cols, rows, client)
	if err != nil {
		s.Logger.Errorf("new ssh connect failed, err: %v", err)
		return
	}
	ws := websocket.NewWSConnect(wsConn, nil)
	defer ssConn.Close()

	quitChan := make(chan bool, 3)
	go ssConn.SendComboOutput(ws.Conn, quitChan)
	go ssConn.ReceiveWsMsg(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	s.Logger.Info("websocket ssh finished")
}
