package page

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"oms/models"
	"oms/routers/wscontrol"
	"oms/transport"
	"strconv"
)

func GetWebsocketSsh(c *gin.Context) {
	idStr := c.Param("id")
	// get pty windows size
	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	id, _ := strconv.Atoi(idStr)
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("upgrade websocket failed, err: %v", err)
	}
	defer wsConn.Close()

	host := models.GetHostById(id)
	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		log.Errorf("transport new client failed, err: %v", err)
	}

	ssConn, err := wscontrol.NewSshConn(cols, rows, client)
	ws := &wscontrol.WSConnect{Conn: wsConn}
	ssConn.SetOutput(ws)
	defer ssConn.Close()

	quitChan := make(chan bool)
	go ssConn.ReceiveWsMsg(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	log.Infoln("websocket ssh finished")
}

func GetWebSocketShell(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		log.Println(err)
		return
	}
	pType := c.Query("type")
	if pType == "" {
		log.Println("shell_ws can't found params type")
		return
	}
	hosts := models.ParseHostList(pType, id)
	wsConn, err1 := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err1 != nil {
		log.Println(err1)
	}
	defer wsConn.Close()

	quitChan := make(chan bool, 2)
	wsClient := wscontrol.NewWebSocketShellClient(wsConn, hosts, quitChan)
	wsClient.Start()

	<-quitChan
	log.Infoln("websocket shell finished")
}
