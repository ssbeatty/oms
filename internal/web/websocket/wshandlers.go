package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/web/payload"
	"oms/pkg/transport"
	"oms/pkg/utils"
	"time"
)

const (
	WSStatusSuccess       = "0"
	WSStatusError         = "-1"
	defaultSSHCMDTimeout  = 120
	DefaultStatusInterval = 2 * time.Second
)

var (
	DefaultSSHCMDTimeout = time.Duration(utils.GetEnvInt("ENV_SSH_CMD_TIMEOUT", defaultSSHCMDTimeout)) * time.Second
)

type Request struct {
	Type  string `json:"type"`
	Id    int    `json:"id"`
	Cmd   string `json:"cmd"`
	CType string `json:"cmd_type"`
	CmdId int    `json:"cmd_id"`
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
		"RESIZE":      w.HandlerResize,
	}
	return w
}

func (w *WSConnect) HandlerResize(conn *websocket.Conn, msg *WsMsg) {
	w.logger.Infof("handler resize recv a message: %s", msg.Body)
	var (
		req = ssh.WindowSize{}
	)

	err := json.Unmarshal(msg.Body, &req)
	if err != nil {
		w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "can not parse payload"))
		return
	}

	w.size = req
}

func (w *WSConnect) HandlerSSHShell(conn *websocket.Conn, msg *WsMsg) {
	w.logger.Infof("handler ssh shell recv a message: %s", msg.Body)
	var (
		execNum int
		req     = &Request{}
		ch      = make(chan *ssh.Result)
	)

	defer close(ch)

	err := json.Unmarshal(msg.Body, req)
	if err != nil {
		w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "can not parse payload"))
		return
	}
	hosts, err := models.ParseHostList(req.Type, req.Id)
	if err != nil || len(hosts) == 0 {
		w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "host empty"))
		return
	}
	defer func() {
		w.logger.Infof("cmd exec success, total: %d, exec: %d", len(hosts), execNum)
		// insert command history to database
		err := models.InsertOrUpdateCommandHistory(req.Cmd)
		if err != nil {
			w.logger.Errorf("create command history error: %v", err)
		}
	}()

	for _, host := range hosts {
		// TODO sudo 由host本身管理
		switch req.CType {
		case ssh.CMDTypePlayer:
			player, err := models.GetPlayBookById(req.CmdId)
			if err != nil {
				w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "playbook not found"))
				return
			}
			cmd := ssh.Command{
				Type:       ssh.CMDTypePlayer,
				Params:     player.Steps,
				Sudo:       true,
				WindowSize: w.size,
			}
			go w.engine.RunCmdWithContext(host, cmd, ch)
		default:
			cmd := ssh.Command{
				Type:       ssh.CMDTypeShell,
				Params:     req.Cmd,
				Sudo:       true,
				WindowSize: w.size,
			}
			go w.engine.RunCmdWithContext(host, cmd, ch)
		}
	}

	for i := 0; i < len(hosts); i++ {
		res := <-ch
		res.Seq = i

		execNum++
		w.WriteMsg(payload.GenerateDataResponse(WSStatusSuccess, "success", res))
	}

	w.WriteMsg(payload.GenerateMsgResponse(
		WSStatusSuccess, fmt.Sprintf("cmd exec success, total: %d, exec: %d", len(hosts), execNum)))
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
				w.WriteMsg(payload.GenerateDataResponse(WSStatusSuccess, "success", resp))
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
		w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "can not parse payload"))
		return
	}
	hosts, err := models.ParseHostList(req.Type, req.Id)
	if err != nil || len(hosts) == 0 {
		w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "parse host array empty"))
		return
	}

	client, err := w.engine.GetSSHManager().NewClient(hosts[0])
	if err != nil {
		w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, fmt.Sprintf("error when new ssh client, id: %d", hosts[0].Id)))
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
		if !*isRunning {
			return
		}
		err := transport.GetAllStats(client, status, nil)
		if err != nil {
			w.WriteMsg(payload.GenerateErrorResponse(WSStatusError, "error when get ssh status"))
			return
		}
		w.WriteMsg(payload.GenerateDataResponse(WSStatusSuccess, "success", status))
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
