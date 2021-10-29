package wscontrol

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"oms/pkg/transport"
	"time"
)

const (
	TaskTickerInterval = 2
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

func (w *WSConnect) InitHandlers() *WSConnect {
	w.handlers = map[string]WsHandler{
		"WS_CMD":      w.HandlerSSHShell,
		"FILE_STATUS": w.HandlerFTaskStatus,
	}
	return w
}

// intChangeToSize 转换单位
func intChangeToSize(s int64) string {
	// 1k 以内
	if s < 1024 {
		return fmt.Sprintf("%.2fb", float64(s))
	} else if s < 1024*1024 {
		return fmt.Sprintf("%.2fkb", float64(s)/1024.0)
	} else if s < 1024*1024*1024 {
		return fmt.Sprintf("%.2fmb", float64(s)/1048576.0)
	} else {
		return fmt.Sprintf("%.2fgb", float64(s)/1073741824.0)
	}
}

func (w *WSConnect) HandlerSSHShell(conn *websocket.Conn, msg *WsMsg) {
	log.Info(msg.Data)
}

func (w *WSConnect) HandlerFTaskStatus(conn *websocket.Conn, msg *WsMsg) {
	log.Infof("handler task status recv a message: %v", msg.Data)
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
					Current: intChangeToSize(task.RSize),
					Total:   intChangeToSize(task.Total),
					Speed:   fmt.Sprintf("%s/s", intChangeToSize((task.RSize-task.CSize)/TaskTickerInterval)),
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
				marshal, _ := json.Marshal(resp)
				err := conn.WriteMessage(websocket.TextMessage, marshal)
				if err != nil {
					log.Errorf("error when write task status, err: %v", err)
				}
			}
		}
	}
}
