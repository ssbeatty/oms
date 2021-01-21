package page

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"log"
	"oms/models"
	"oms/routers/wscontrol"
	"oms/ssh"
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
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		log.Println(err)
	}
	ssConn, err := wscontrol.NewSshConn(cols, rows, client.SSHClient)
	defer ssConn.Close()
	quitChan := make(chan bool, 3)
	var logBuff = new(bytes.Buffer)
	go ssConn.ReceiveWsMsg(wsConn, logBuff, quitChan)
	go ssConn.SendComboOutput(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	log.Println("websocket ssh finished")
}

func GetWebSocketShell(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		log.Println(err)
	}
	pType := c.Query("type")
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
