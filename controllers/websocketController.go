package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
	"net/http"
)

type WebSocketController struct {
	beego.Controller
}

func (this *WebSocketController) Get() {
	// Upgrade from http request to WebSocket.
	ws, err := websocket.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(this.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		beego.Error("Cannot setup WebSocket connection:", err)
		return
	}

	// Message receive loop.
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			return
		}
		fmt.Println(string(p))
	}
}
