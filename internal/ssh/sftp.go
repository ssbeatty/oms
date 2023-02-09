package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"oms/internal/models"
	"oms/pkg/transport"
	"oms/pkg/utils"
	"os"
	"sync/atomic"
	"time"
)

const (
	TaskTickerInterval = 2
	DefaultBlockSize   = 65535
	FileTaskRunning    = "running"
	FileTaskDone       = "done"
	FileTaskFailed     = "failed"
	FileTaskCancel     = "cancel"
)

type TaskItem struct {
	Status   string
	Cancel   context.CancelFunc
	Total    int64 // 文件总字节
	RSize    int64 // 已传输字节
	CSize    int64 // 当前字节
	FileName string
	Host     string
}

type TempFile struct {
	Path string
	Name string
	Size int
	Num  int32
}

type FTaskResp struct {
	Id      string  `json:"id"`
	File    string  `json:"file"`
	Dest    string  `json:"dest"`
	Speed   string  `json:"speed"`
	Current string  `json:"current"`
	Total   string  `json:"total"`
	Status  string  `json:"status"`
	Percent float32 `json:"percent"`
}

func (m *Manager) UpdateTaskStatus(task *TaskItem, status string) {
	task.Status = status
	m.notify <- true
}

func (m *Manager) CancelTask(key string) {
	if val, ok := m.fileList.Load(key); ok {
		if task, is := val.(*TaskItem); is {
			task.Cancel()
		}
	}
}

func (m *Manager) doNotifyFileTaskList() {
	ticker := time.NewTicker(TaskTickerInterval * time.Second)

	doSendFileTaskListInfo := func() {
		if m.fileList.Length() == 0 {
			return
		}
		var resp []FTaskResp
		m.fileList.Range(func(key, value interface{}) bool {
			var percent float32
			task := value.(*TaskItem)
			if task.Total == 0 {
				percent = 100
			} else {
				percent = float32(task.RSize) * 100.0 / float32(task.Total)
			}
			resp = append(resp, FTaskResp{
				File:    task.FileName,
				Dest:    task.Host,
				Current: utils.IntChangeToSize(task.RSize),
				Total:   utils.IntChangeToSize(task.Total),
				Speed:   fmt.Sprintf("%s/s", utils.IntChangeToSize((task.RSize-task.CSize)/2)),
				Status:  task.Status,
				Percent: percent,
				Id:      utils.HashSha1(fmt.Sprintf("%s/%s", task.Host, task.FileName)),
			})
			task.CSize = task.RSize
			if task.Status == FileTaskDone || task.Status == FileTaskFailed || task.Status == FileTaskCancel {
				m.fileList.Delete(key)
			}
			return true
		})
		m.subClients.Range(func(key, value interface{}) bool {
			ch := value.(chan []FTaskResp)
			ch <- resp

			return true
		})
	}
	for {
		select {
		case <-ticker.C:
			doSendFileTaskListInfo()
		case <-m.notify:
			doSendFileTaskListInfo()
		}
	}
}

func (m *Manager) RegisterFileListSub(key string, notifyCh chan []FTaskResp) {
	m.subClients.Store(key, notifyCh)
}

func (m *Manager) RemoveFileListSub(key string) {
	if val, ok := m.subClients.LoadAndDelete(key); ok {
		close(val.(chan []FTaskResp))
	}
}

func (m *Manager) UploadFileStream(c *transport.Client, remote string, addr string, tmp *TempFile, parentCtx context.Context) {
	// 引用计数 -1
	defer func() {
		atomic.AddInt32(&tmp.Num, -1)
		if tmp.Num <= 0 {
			os.Remove(tmp.Path)
		}
	}()

	ctx, cancel := context.WithCancel(parentCtx)

	var (
		task = &TaskItem{
			Cancel:   cancel,
			Total:    int64(tmp.Size),
			FileName: tmp.Name,
			Status:   FileTaskRunning,
			Host:     addr,
		}
		doCancel bool
	)

	// 重复文件跳过
	key := fmt.Sprintf("%s/%s", addr, tmp.Name)
	if val, ok := m.fileList.Load(key); ok {
		if val.(*TaskItem).Status == FileTaskRunning {
			return
		} else {
			m.fileList.Store(key, task)
		}
	} else {
		m.fileList.Store(key, task)
	}

	// 打开tmp文件
	file, err := os.OpenFile(tmp.Path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		m.UpdateTaskStatus(task, FileTaskFailed)
		m.logger.Errorf("UploadFileStream error when open tmp file, err: %v", err)
		return
	}

	remoteFile, remoteDir := utils.ParseUploadPath(remote, tmp.Name)
	if _, err := c.GetSftpClient().Stat(remoteDir); err != nil {
		if err := c.MkdirAll(remoteDir); err != nil {
			m.UpdateTaskStatus(task, FileTaskFailed)
			m.logger.Errorf("error when sftp create dirs, err: %v", err)
			return
		}
	}
	r, err := c.GetSftpClient().Create(remoteFile)
	if err != nil {
		m.logger.Errorf("error when sftp create file, err: %v", err)
		m.UpdateTaskStatus(task, FileTaskFailed)
		return
	}

	defer func() {
		m.logger.Debugf("upload file goroutine exit.")
		_ = file.Close()
		_ = r.Close()
		if doCancel {
			m.UpdateTaskStatus(task, FileTaskCancel)
			return
		}
		// 遇到关闭连接的情况
		if task.Status != FileTaskDone {
			m.UpdateTaskStatus(task, FileTaskFailed)
		}
	}()

	go func() {
		select {
		case <-parentCtx.Done():
			_ = file.Close()
			return
		case <-ctx.Done():
			_ = file.Close()
			doCancel = true
			return
		}
	}()

	readBuf := make([]byte, DefaultBlockSize)
	size := tmp.Size
	for size > 0 {
		n, err := file.Read(readBuf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return
		}
		task.RSize += int64(n)
		size -= n
		_, err = r.Write(readBuf[:n])
		if err != nil {
			m.logger.Errorf("error when write to sftp file, err: %v", err)
			return
		}
	}
	m.UpdateTaskStatus(task, FileTaskDone)

	m.logger.Debugf("file: %s, size: %d, status: %s", task.FileName, task.RSize, task.Status)
}

// NewClientWithSftp 创建新的ssh client并创建sftp客户端
func (m *Manager) NewClientWithSftp(host *models.Host) (*transport.Client, error) {
	client, err := m.NewClient(host)
	if err != nil {
		return nil, err
	}
	err = client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}
