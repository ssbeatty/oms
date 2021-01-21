package wscontrol

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"oms/models"
)

type WebSocketShellClient struct {
	WsConn      *websocket.Conn
	Hosts       []*models.Host
	chanSshResp chan *models.Result
	chanQuit    chan bool
}

type WsResponse struct {
	Name   string `json:"name"`
	Msg    string `json:"msg"`
	Err    string `json:"err"`
	Status bool   `json:"status"`
}

func NewWebSocketShellClient(WsConn *websocket.Conn, Hosts []*models.Host, chanQuit chan bool) *WebSocketShellClient {
	return &WebSocketShellClient{
		WsConn:      WsConn,
		Hosts:       Hosts,
		chanSshResp: make(chan *models.Result),
		chanQuit:    chanQuit,
	}
}

func (ws WebSocketShellClient) RecvWsMsg() {
	defer ws.setQuit()
	for {
		select {
		case <-ws.chanQuit:
			log.Println("RecvWsMsg Recv quit chan, exit!")
			return

		default:
			_, wsData, err := ws.WsConn.ReadMessage()
			if err != nil {
				log.Println("reading webSocket message failed")
				return
			}
			for idx, _ := range ws.Hosts {
				go models.RunCmdOneAsync(ws.Hosts[idx], string(wsData), ws.chanSshResp)
			}
		}
	}
}

func (ws WebSocketShellClient) WriteWsMsg() {
	for {
		select {
		case <-ws.chanQuit:
			log.Println("WriteWsMsg Recv quit chan, exit!")
			return
		case sshResp := <-ws.chanSshResp:
			// TODO err msg
			wsResp, _ := json.Marshal(WsResponse{Name: sshResp.HostName, Status: sshResp.Status, Msg: sshResp.Msg, Err: ""})
			if err := ws.WsConn.WriteMessage(websocket.TextMessage, wsResp); err != nil {
				log.Printf("Ws WriteMessage err: %v", err)
			}
		}
	}
}

func (ws WebSocketShellClient) Start() {
	go ws.RecvWsMsg()
	go ws.WriteWsMsg()
}

func (ws WebSocketShellClient) setQuit() {
	close(ws.chanQuit)
}
