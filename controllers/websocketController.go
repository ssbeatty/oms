package controllers

import (
	"bytes"
	"github.com/astaxie/beego"
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

type WebSocketController struct {
	beego.Controller
}

func (c *WebSocketController) Get() {
	idStr := c.Ctx.Input.Param(":id")
	cols, _ := strconv.Atoi(c.Input().Get("cols"))
	rows, _ := strconv.Atoi(c.Input().Get("rows"))
	id, _ := strconv.Atoi(idStr)
	logger.Logger.Println(id, rows, cols)
	wsConn, err := upGrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
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
