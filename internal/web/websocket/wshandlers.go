package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"oms/internal/ssh"
	"oms/internal/web/payload"
	"oms/pkg/transport"
	"time"
)

const (
	WSStatusSuccess       = "0"
	WSStatusError         = "-1"
	SSHTimeDeadline       = 30 * time.Second
	DefaultStatusInterval = 2 * time.Second
)

type Request struct {
	Type string `json:"type"`
	Id   int    `json:"id"`
	Cmd  string `json:"cmd"`
}

type HostStatusRequest struct {
	Request
	Interval int `json:"interval"`
}

func (w *WSConnect) InitHandlers() *WSConnect {
	w.handlers = map[string]WsHandler{
		"WS_CMD":      w.HandlerSSHShell,
		"FILE_STATUS": w.HandlerFTaskStatus,
		"HOST_STATUS": w.HandlerHostStatus,
	}
	return w
}

func (w *WSConnect) HandlerSSHShell(conn *websocket.Conn, msg *WsMsg) {
	w.logger.Infof("handler ssh shell recv a message: %s", msg.Body)
	req := &Request{}
	ch := make(chan interface{})

	err := json.Unmarshal(msg.Body, req)
	if err != nil {
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, "can not parse payload", nil))
		return
	}
	hosts := w.engine.ParseHostList(req.Type, req.Id)
	if len(hosts) == 0 {
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, "host empty", nil))
		return
	}
	for _, host := range hosts {
		// TODO sudo 由host本身管理
		go w.engine.RunCmdWithContext(host, req.Cmd, true, ch)
	}

	for i := 0; i < len(hosts); i++ {
		res := <-ch
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusSuccess, "success", res))
	}

	close(ch)
}

func (w *WSConnect) HandlerFTaskStatus(conn *websocket.Conn, msg *WsMsg) {
	w.logger.Infof("handler task status recv a message: %s", msg.Body)

	notifyCh := make(chan []ssh.FTaskResp)
	key, err := uuid.NewUUID()
	if err != nil {
		w.logger.Errorf("create uuid error: %v.", err)
		return
	}
	w.engine.GetSSHManager().RegisterFileListSub(key.String(), notifyCh)
	defer w.engine.GetSSHManager().RemoveFileListSub(key.String())

	var quit chan struct{}
	if msg.Event == EventConnect {
		quit = w.addSubscribe(msg.Type)
	}

	for {
		select {
		case <-quit:
			w.logger.Debug("file task status exit because cancel sub.")
			return
		case <-w.closer:
			w.logger.Debug("file task status exit.")
			return
		case resp := <-notifyCh:
			if len(resp) > 0 {
				w.WriteMsg(payload.GenerateResponsePayload(WSStatusSuccess, "success", resp))
			}
		}
	}
}

func (w *WSConnect) HandlerHostStatus(conn *websocket.Conn, msg *WsMsg) {
	w.logger.Infof("handler host status recv a message: %s", msg.Body)

	req := &HostStatusRequest{}
	var status = transport.NewStatus()

	err := json.Unmarshal(msg.Body, req)
	if err != nil {
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, "can not parse payload", nil))
		return
	}
	hosts := w.engine.ParseHostList(req.Type, req.Id)
	if len(hosts) == 0 {
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, "parse host array empty", nil))
		return
	}

	client, err := w.engine.GetSSHManager().NewClient(hosts[0])
	if err != nil {
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, fmt.Sprintf("error when new ssh client, id: %d", hosts[0].Id), nil))
		return
	}

	var interval time.Duration
	if req.Interval < int(DefaultStatusInterval)/1e9 {
		interval = DefaultStatusInterval
	} else {
		interval = time.Duration(req.Interval) * time.Second
	}

	var isRunning = true
	defer func() { isRunning = false }()

	var sendHostMsg = func(isRunning *bool) {
		err := transport.GetAllStats(client, status, nil)
		if err != nil && *isRunning {
			w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, fmt.Sprintf("error when get ssh status"), nil))
			return
		}
		if *isRunning {
			w.WriteMsg(payload.GenerateResponsePayload(WSStatusSuccess, "success", status))
		}
	}

	go sendHostMsg(&isRunning)

	ticker := time.NewTicker(interval)

	var quit chan struct{}
	if msg.Event == EventConnect {
		quit = w.addSubscribe(msg.Type)
	}

	for {
		select {
		case <-quit:
			w.logger.Debug("host status exit because cancel sub.")
			return
		case <-ticker.C:
			go sendHostMsg(&isRunning)
		case <-w.closer:
			w.logger.Info("host status loop return.")
			return
		}
	}
}
