package page

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"log"
	"oms/models"
	"oms/routers/wscontrol"
	"oms/transport"
	"strconv"
)

func GetWebsocketSsh(c *gin.Context) {
	idStr := c.Param("id")
	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	id, _ := strconv.Atoi(idStr)
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
	}
	defer wsConn.Close()
	host := models.GetHostById(id)
	ws := &wscontrol.WSConnect{Conn: wsConn}
	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		log.Println(err)
	}
	ssConn, err := wscontrol.NewSshConn(cols, rows, client)

	ssConn.Session.SSHSession.Stderr = ws
	ssConn.Session.SSHSession.Stdout = ws
	defer ssConn.Close()
	quitChan := make(chan bool)
	var logBuff = new(bytes.Buffer)
	go ssConn.ReceiveWsMsg(wsConn, logBuff, quitChan)
	//go ssConn.SendComboOutput(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	log.Println("websocket ssh finished")
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
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
	}
	defer wsConn.Close()

	quitChan := make(chan bool, 2)
	wsClient := wscontrol.NewWebSocketShellClient(wsConn, hosts, quitChan)
	wsClient.Start()

	<-quitChan
	log.Println("websocket shell finished")
}
