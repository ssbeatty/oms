package wscontrol

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WsHandler func(conn *websocket.Conn, msg *WsMsg)

type WSConnect struct {
	*websocket.Conn
	onMessage func(b []byte)
	handlers  map[string]WsHandler
	closer    chan struct{}
}

type WsMsg struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func (w *WSConnect) Write(p []byte) (int, error) {
	err := w.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func NewWSConnect(conn *websocket.Conn) *WSConnect {
	c := &WSConnect{Conn: conn}
	c.onMessage = c.OnMessage
	return c
}

func (w *WSConnect) InitHandlers() {
	w.handlers = map[string]WsHandler{
		"WS_CMD": HandlerSSHShell,
	}
}

func (w *WSConnect) Serve() error {
	w.mange()
	return nil
}

func (w *WSConnect) mange() {
	for {
		select {
		case <-w.closer:
			log.Debug("ws connect mange close")
			return
		default:
			_, b, err := w.ReadMessage()
			if err != nil {
				log.Debugf("read message from wsconnect failed, err: %v", err)
				continue
			}
			if w.onMessage != nil {
				w.onMessage(b)
			}
		}
	}
}

func (w *WSConnect) Close() error {
	w.closer <- struct{}{}
	return w.Conn.Close()
}

func (w *WSConnect) getHandler(t string) WsHandler {
	handler, ok := w.handlers[t]
	if ok {
		return handler
	} else {
		return nil
	}
}

func (w *WSConnect) OnMessage(buf []byte) {
	msg := &WsMsg{}
	err := json.Unmarshal(buf, msg)
	if err != nil {
		log.Debugf("onmeesage unmarshal failed, err: %v", err)
	}
	handler := w.getHandler(msg.Type)
	if handler != nil {
		handler(w.Conn, msg)
	}
}

func HandlerSSHShell(conn *websocket.Conn, msg *WsMsg) {
	log.Info(msg.Data)
}
