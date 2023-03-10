package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	wsl "github.com/gorilla/websocket"
	"github.com/ssbeatty/oms/internal/models"
	"github.com/ssbeatty/oms/internal/web/websocket"
	"github.com/ssbeatty/oms/web"
	"net"
	"net/http"
	"strconv"
)

const IndexPage = "omsUI/dist/index.html"

func (s *Service) GetIndexPage(c *gin.Context) {
	bytes, err := web.EmbeddedFiles.ReadFile(IndexPage)
	if err != nil {
		fmt.Println("err", err)
	}
	c.Data(http.StatusOK, "", bytes)
}

var upGrader = wsl.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GetWebsocketIndex default websocket router
func (s *Service) GetWebsocketIndex(c *gin.Context) {
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.Logger.Errorf("upgrade websocket failed, err: %v", err)
	}
	ws := websocket.NewWSConnect(wsConn, s).InitHandlers()
	ws.Serve()
}

// GetWebsocketSSH func websocket ssh
func (s *Service) GetWebsocketSSH(c *gin.Context) {
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

	quitChan := make(chan struct{}, 3)
	go ssConn.SendComboOutput(ws.Conn, quitChan)
	go ssConn.ReceiveWsMsg(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	s.Logger.Info("websocket ssh finished")
}

// GetWebsocketVNC func websocket vnc proxy
// https://github.com/novnc/websockify-other
func (s *Service) GetWebsocketVNC(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
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

	vnc, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host.Addr, host.VNCPort))
	if err != nil {
		s.Logger.Errorf("failed to bind to the VNC Server: %s", err)
		return
	}

	quitChan := make(chan struct{})
	forward := websocket.NewVNCForward(wsConn, vnc, s.Logger, quitChan)
	defer forward.Close()

	forward.Serve()
	s.Logger.Info("websocket vnc finished")
}
