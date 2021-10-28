package wscontrol

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WsHandler func(conn *websocket.Conn, msg *WsMsg)

type WSConnect struct {
	*websocket.Conn
	handlers map[string]WsHandler
	closer   chan bool
}

type WsMsg struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewWSConnect(conn *websocket.Conn) *WSConnect {
	c := &WSConnect{Conn: conn}
	return c
}

func (w *WSConnect) Serve() {
	w.mange()
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
			msg := &WsMsg{}
			err = json.Unmarshal(b, msg)
			if err != nil {
				log.Debugf("on message unmarshal failed, err: %v", err)
			}
			handler := w.getHandler(msg.Type)
			if handler != nil {
				handler(w.Conn, msg)
			}
		}
	}
}

func (w *WSConnect) Close() error {
	w.closer <- true
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
