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
	WSStatusSuccess = "0"
	WSStatusError   = "-1"
	SSHTimeDeadline = 30 * time.Second
)

type Request struct {
	Type string `json:"type"`
	Id   int    `json:"id"`
	Cmd  string `json:"cmd"`
}

func (w *WSConnect) InitHandlers() *WSConnect {
	w.handlers = map[string]WsHandler{
		"WS_CMD":      w.HandlerSSHShell,
		"FILE_STATUS": w.HandlerFTaskStatus,
		"HOST_STATUS": w.HandlerHostStatus,
	}
	return w
}

func (w *WSConnect) HandlerSSHShell(conn *websocket.Conn, msg []byte) {
	w.logger.Infof("handler ssh shell recv a message: %s", msg)
	req := &Request{}
	ch := make(chan interface{})

	err := json.Unmarshal(msg, req)
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

func (w *WSConnect) HandlerFTaskStatus(conn *websocket.Conn, msg []byte) {
	w.logger.Infof("handler task status recv a message: %s", msg)

	// 一个连接只能有一个订阅
	const fTaskFlag = "f_task_existed"
	val, ok := w.LoadCache(fTaskFlag)
	if ok && val.(bool) {
		w.WriteMsg(payload.GenerateResponsePayload(WSStatusError, "subscription already exists", nil))
		return
	} else {
		w.StoreCache(fTaskFlag, true)
	}

	notifyCh := make(chan []ssh.FTaskResp)
	key, err := uuid.NewUUID()
	if err != nil {
		w.logger.Errorf("create uuid error: %v.", err)
		return
	}
	w.engine.GetSSHManager().RegisterFileListSub(key.String(), notifyCh)
	defer w.engine.GetSSHManager().RemoveFileListSub(key.String())

	for {
		select {
		case <-w.closer:
			w.logger.Debug("file task status exit.")
			return
		case resp := <-notifyCh:
			if len(resp) > 0 {
				w.WriteMsg(payload.GenerateResponsePayload(WSStatusSuccess, "success", nil))
			}
		}
	}
}

func (w *WSConnect) HandlerHostStatus(conn *websocket.Conn, msg []byte) {
	w.logger.Infof("handler host status recv a message: %s", msg)
	req := &Request{}
	var status *transport.Stats

	val, ok := w.LoadCache("status")
	if ok {
		status = val.(*transport.Stats)
	} else {
		status = transport.NewStatus()
	}

	err := json.Unmarshal(msg, req)
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
	}
	transport.GetAllStats(client, status, nil)
	w.StoreCache("status", status)
	w.WriteMsg(payload.GenerateResponsePayload(WSStatusSuccess, "success", nil))
}
