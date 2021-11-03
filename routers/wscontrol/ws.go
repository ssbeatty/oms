package wscontrol

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"oms/pkg/transport"
)

type WsHandler func(conn *websocket.Conn, msg []byte)

type WSConnect struct {
	*websocket.Conn
	handlers map[string]WsHandler
	closer   chan bool
	status   *transport.Stats
}

type WsMsg struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewWSConnect(conn *websocket.Conn) *WSConnect {
	c := &WSConnect{
		Conn:   conn,
		closer: make(chan bool),
	}
	return c
}

func (w *WSConnect) Serve() {
	go w.mange()
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
				_ = w.Close()
			}
			msg := &WsMsg{}
			err = json.Unmarshal(b, msg)
			if err != nil {
				log.Debugf("on message unmarshal failed, err: %v", err)
			}
			handler := w.getHandler(msg.Type)
			if handler != nil {
				data, _ := json.Marshal(msg.Data)
				go handler(w.Conn, data)
			}
		}
	}
}

func (w *WSConnect) Write(p []byte) (int, error) {
	err := w.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *WSConnect) WriteMsg(msg interface{}) {
	marshal, _ := json.Marshal(msg)
	err := w.WriteMessage(websocket.TextMessage, marshal)
	if err != nil {
		log.Errorf("error when write msg, err: %v", err)
	}
}

func (w *WSConnect) Close() error {
	defer close(w.closer)
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
