package web

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"oms/internal/ssh"
	"oms/internal/utils"
	"oms/pkg/transport"
	"time"
)

const (
	TaskTickerInterval = 2
	WSStatusSuccess    = "0"
	WSStatusError      = "-1"
	SSHTimeDeadline    = 30 * time.Second
)

type FTaskResp struct {
	File    string  `json:"file"`
	Dest    string  `json:"dest"`
	Speed   string  `json:"speed"`
	Current string  `json:"current"`
	Total   string  `json:"total"`
	Status  string  `json:"status"`
	Percent float32 `json:"percent"`
}

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
	ch := make(chan *Result)

	err := json.Unmarshal(msg, req)
	if err != nil {
		w.WriteMsg(Response{Code: WSStatusError, Msg: "can not parse payload"})
		return
	}
	hosts := w.engine.ParseHostList(req.Type, req.Id)
	if len(hosts) == 0 {
		w.WriteMsg(Response{Code: WSStatusError, Msg: "host empty"})
		return
	}
	for _, host := range hosts {
		// TODO sudo 由host本身管理
		go w.engine.RunCmdWithContext(host, req.Cmd, false, ch)
	}

	for i := 0; i < len(hosts); i++ {
		res := <-ch
		w.WriteMsg(Response{Code: WSStatusSuccess, Msg: "success", Data: res})
	}

	close(ch)
}

func (w *WSConnect) HandlerFTaskStatus(conn *websocket.Conn, msg []byte) {
	w.logger.Infof("handler task status recv a message: %s", msg)
	ticker := time.NewTicker(TaskTickerInterval * time.Second)

	// 一个连接只能有一个订阅
	const fTaskFlag = "f_task_existed"
	val, ok := w.LoadCache(fTaskFlag)
	if ok && val.(bool) {
		return
	} else {
		w.StoreCache(fTaskFlag, true)
	}

	var sendCurrentFileStatus = func() {
		var resp []FTaskResp
		w.engine.sshManager.GetFileList().Range(func(key, value interface{}) bool {
			task := value.(*ssh.TaskItem)
			percent := float32(task.RSize) * 100.0 / float32(task.Total)
			resp = append(resp, FTaskResp{
				File:    task.FileName,
				Dest:    task.Host,
				Current: utils.IntChangeToSize(task.RSize),
				Total:   utils.IntChangeToSize(task.Total),
				Speed:   fmt.Sprintf("%s/s", utils.IntChangeToSize((task.RSize-task.CSize)/TaskTickerInterval)),
				Status:  task.Status,
				Percent: percent,
			})
			task.CSize = task.RSize
			if task.Status == ssh.FileTaskDone || task.Status == ssh.FileTaskFailed {
				w.engine.sshManager.GetFileList().Delete(key)
			}
			return true
		})
		if len(resp) > 0 {
			w.WriteMsg(Response{Code: WSStatusSuccess, Msg: "success", Data: resp})
		}
	}

	sendCurrentFileStatus()

	for {
		select {
		case <-w.closer:
			w.logger.Debug("file task status exit.")
			return
		case <-ticker.C:
			sendCurrentFileStatus()
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
		w.WriteMsg(Response{Code: WSStatusError, Msg: "can not parse payload"})
		return
	}
	hosts := w.engine.ParseHostList(req.Type, req.Id)
	if len(hosts) == 0 {
		w.WriteMsg(Response{Code: WSStatusError, Msg: "parse host array empty"})
		return
	}

	client, err := w.engine.sshManager.NewClient(hosts[0].Addr, hosts[0].Port, hosts[0].User, hosts[0].PassWord, []byte(hosts[0].KeyFile))
	if err != nil {
		w.WriteMsg(Response{Code: WSStatusError, Msg: fmt.Sprintf("error when new ssh client, id: %d", hosts[0].Id)})
	}
	transport.GetAllStats(client.GetSSHClient(), status, nil)
	w.StoreCache("status", status)
	w.WriteMsg(Response{Code: WSStatusSuccess, Msg: "success", Data: status})
}
