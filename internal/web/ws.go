package web

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"oms/pkg/logger"
	"sync"
)

type WsHandler func(conn *websocket.Conn, msg []byte)

type WSConnect struct {
	*websocket.Conn
	engine   *Service
	handlers map[string]WsHandler
	closer   chan bool
	once     sync.Once
	tmp      sync.Map
	logger   *logger.Logger
}

type WsMsg struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewWSConnect(conn *websocket.Conn, engine *Service) *WSConnect {
	c := &WSConnect{
		Conn:   conn,
		engine: engine,
		closer: make(chan bool),
		once:   sync.Once{},
		tmp:    sync.Map{},
		logger: logger.NewLogger("websocket"),
	}
	return c
}

func (w *WSConnect) StoreCache(key, val interface{}) {
	w.tmp.Store(key, val)
}

func (w *WSConnect) LoadCache(key interface{}) (interface{}, bool) {
	if r, ok := w.tmp.Load(key); ok {
		return r, true
	}
	return nil, false
}

func (w *WSConnect) Serve() {
	go w.mange()
}

func (w *WSConnect) mange() {
	for {
		select {
		case <-w.closer:
			w.logger.Debug("ws connect mange close")
			return
		default:
			_, b, err := w.ReadMessage()
			if err != nil {
				w.logger.Debugf("read message from wsconnect failed, err: %v", err)
				_ = w.Close()
			}
			msg := &WsMsg{}
			err = json.Unmarshal(b, msg)
			if err != nil {
				w.logger.Debugf("on message unmarshal failed, err: %v", err)
			}
			handler := w.getHandler(msg.Type)
			if handler != nil {
				data, _ := json.Marshal(msg.Data)
				go handler(w.Conn, data)
			}
		}
	}
}

func (w *WSConnect) WriteMsg(msg interface{}) {
	marshal, _ := json.Marshal(msg)
	err := w.WriteMessage(websocket.TextMessage, marshal)
	if err != nil {
		w.logger.Errorf("error when write msg, err: %v", err)
	}
}

func (w *WSConnect) Close() error {
	w.once.Do(func() {
		close(w.closer)
	})
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
