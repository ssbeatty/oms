package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"oms/internal/ssh"
	"oms/pkg/logger"
	"oms/pkg/transport"
	"sync"
)

const (
	EventCancel  = "cancel"
	EventConnect = "connect"
)

type WsHandler func(conn *websocket.Conn, msg *WsMsg)

type WSConnect struct {
	*websocket.Conn
	mu             sync.Mutex
	engine         WebService
	handlers       map[string]WsHandler
	closer         chan struct{}
	once           sync.Once
	tmp            sync.Map
	logger         *logger.Logger
	size           ssh.WindowSize
	existSubscribe map[string]chan struct{}
}

type WsMsg struct {
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
	Event string      `json:"event"`
	Body  []byte      `json:"-"`
}

func NewWSConnect(conn *websocket.Conn, engine WebService) *WSConnect {
	c := &WSConnect{
		Conn:           conn,
		engine:         engine,
		closer:         make(chan struct{}),
		once:           sync.Once{},
		tmp:            sync.Map{},
		logger:         logger.NewLogger("websocket"),
		existSubscribe: make(map[string]chan struct{}),
		mu:             sync.Mutex{},
		size:           ssh.WindowSize{Cols: transport.DefaultPtyCols, Rows: transport.DefaultPtyRows},
	}
	return c
}

func (w *WSConnect) subscribeExisted(key string) (chan struct{}, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if sub, exist := w.existSubscribe[key]; exist {
		return sub, true
	}
	return nil, false
}

func (w *WSConnect) cancelSubscribe(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if sub, exist := w.existSubscribe[key]; exist {
		close(sub)
		delete(w.existSubscribe, key)
	}

}

func (w *WSConnect) addSubscribe(key string) chan struct{} {
	if quit, exist := w.subscribeExisted(key); exist {
		return quit
	}
	quit := make(chan struct{})

	w.mu.Lock()
	w.existSubscribe[key] = quit
	w.mu.Unlock()

	return quit
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
			if msg.Event == EventCancel {
				w.cancelSubscribe(msg.Type)
				continue
			}
			handler := w.getHandler(msg.Type)
			if handler != nil {
				msg.Body, _ = json.Marshal(msg.Data)
				go handler(w.Conn, msg)
			}
		}
	}
}

func (w *WSConnect) WriteMsg(msg interface{}) {
	// 不能并发的往websocket写入消息
	w.mu.Lock()
	defer w.mu.Unlock()

	marshal, err := json.Marshal(msg)
	if err != nil {
		w.logger.Errorf("error when marshal websocket msg payload: %v", err)
		return
	}
	err = w.WriteMessage(websocket.TextMessage, marshal)
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
