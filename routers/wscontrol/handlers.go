package wscontrol

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func (w *WSConnect) InitHandlers() *WSConnect {
	w.handlers = map[string]WsHandler{
		"WS_CMD": HandlerSSHShell,
	}
	return w
}

func HandlerSSHShell(conn *websocket.Conn, msg *WsMsg) {
	log.Info(msg.Data)
}
