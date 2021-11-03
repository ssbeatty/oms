package wscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"oms/models"
	"oms/pkg/transport"
	"oms/pkg/utils"
	"sync"
	"time"
)

const (
	TaskTickerInterval = 2
	RequestTypeHost    = "host"
	RequestTypeGroup   = "group"
	RequestTypeTag     = "tag"
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

type Response struct {
	ErrorCode int         `json:"code"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data,omitempty"`
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
	log.Infof("handler ssh shell recv a message: %s", msg)
	req := &Request{}
	ch := make(chan *models.Result)

	err := json.Unmarshal(msg, req)
	if err != nil {
		w.WriteMsg(Response{ErrorCode: -1, Msg: "can not parse payload"})
		return
	}
	hosts := models.ParseHostList(req.Type, req.Id)
	if len(hosts) == 0 {
		w.WriteMsg(Response{ErrorCode: -2, Msg: "host empty"})
		return
	}
	for _, host := range hosts {
		ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(SSHTimeDeadline))
		// TODO sudo 由host本身管理
		go models.RunCmdWithContext(host, req.Cmd, true, ch, ctx)
	}

	for i := 0; i < len(hosts); i++ {
		res := <-ch
		w.WriteMsg(Response{ErrorCode: 0, Msg: "success", Data: res})
	}

	close(ch)
}

func (w *WSConnect) HandlerFTaskStatus(conn *websocket.Conn, msg []byte) {
	log.Infof("handler task status recv a message: %s", msg)
	ticker := time.NewTicker(TaskTickerInterval * time.Second)

	for {
		select {
		case <-w.closer:
			log.Debug("file task status exit.")
			return
		case <-ticker.C:
			var resp []FTaskResp
			transport.CurrentFiles.Range(func(key, value interface{}) bool {
				task := value.(*transport.TaskItem)
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
				if task.Status == transport.TaskDone || task.Status == transport.TaskFailed {
					transport.CurrentFiles.Delete(key)
				}
				return true
			})
			if len(resp) > 0 {
				w.WriteMsg(Response{ErrorCode: 0, Msg: "success", Data: resp})
			} else {
				w.WriteMsg(Response{ErrorCode: -1, Msg: "file task empty"})
				return
			}
		}
	}
}

func (w *WSConnect) HandlerHostStatus(conn *websocket.Conn, msg []byte) {
	log.Infof("handler host status recv a message: %s", msg)
	req := &Request{}
	w.status = transport.NewStatus()

	err := json.Unmarshal(msg, req)
	if err != nil {
		w.WriteMsg(Response{ErrorCode: -1, Msg: "can not parse payload"})
		return
	}
	hosts := models.ParseHostList(req.Type, req.Id)
	if len(hosts) == 0 {
		w.WriteMsg(Response{ErrorCode: -2, Msg: "parse host array empty"})
		return
	}
	wg := sync.WaitGroup{}

	var Result []transport.Stats
	for _, host := range hosts {
		wg.Add(1)
		client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
		if err != nil {
			w.WriteMsg(Response{ErrorCode: -3, Msg: fmt.Sprintf("error when new ssh client, id: %d", host.Id)})
		}
		transport.GetAllStats(client.GetSSHClient(), w.status, &wg)
		Result = append(Result, *w.status)
	}
	wg.Wait()
	w.WriteMsg(Response{ErrorCode: 0, Msg: "success", Data: Result})
}
