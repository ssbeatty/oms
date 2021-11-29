package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"oms/internal/models"
	"oms/internal/utils"
	"oms/pkg/transport"
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
)

type TaskItem struct {
	Status   string
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

func (m *Manager) doNotifyFileTaskList() {
	ticker := time.NewTicker(TaskTickerInterval * time.Second)

	doSendFileTaskListInfo := func() {
		if m.fileList.Length() == 0 {
			return
		}
		var resp []FTaskResp
		m.fileList.Range(func(key, value interface{}) bool {
			task := value.(*TaskItem)
			percent := float32(task.RSize) * 100.0 / float32(task.Total)
			resp = append(resp, FTaskResp{
				File:    task.FileName,
				Dest:    task.Host,
				Current: utils.IntChangeToSize(task.RSize),
				Total:   utils.IntChangeToSize(task.Total),
				Speed:   fmt.Sprintf("%s/s", utils.IntChangeToSize((task.RSize-task.CSize)/2)),
				Status:  task.Status,
				Percent: percent,
			})
			task.CSize = task.RSize
			if task.Status == FileTaskDone || task.Status == FileTaskFailed {
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

// UploadFileOneAsync 上传文件并将addr/filename维护到file task pool
func (m *Manager) UploadFileOneAsync(c *transport.Client, fileH *multipart.FileHeader, remote string, addr string, filename string) {
	task := &TaskItem{
		Total:    fileH.Size,
		FileName: fileH.Filename,
		Status:   FileTaskRunning,
		Host:     addr,
	}

	file, err := fileH.Open()
	if err != nil {
		m.logger.Errorf("error when open multipart file, err: %v", err)
		m.UpdateTaskStatus(task, FileTaskFailed)
		return
	}

	remoteFile, remoteDir := utils.ParseUploadPath(remote, fileH.Filename)
	if _, err := c.GetSftpClient().Stat(remoteDir); err != nil {
		if err := c.MkdirAll(remoteDir); err != nil {
			m.UpdateTaskStatus(task, FileTaskFailed)
			m.logger.Errorf("error when sftp create dirs, err: %v", err)
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
		// 遇到关闭连接的情况
		if task.Status != FileTaskDone {
			m.UpdateTaskStatus(task, FileTaskFailed)
		}
	}()

	key := fmt.Sprintf("%s/%s", addr, filename)
	if val, ok := m.fileList.Load(key); ok {
		if val.(*TaskItem).Status == FileTaskRunning {
			return
		} else {
			m.fileList.Store(key, task)
		}
	} else {
		m.fileList.Store(key, task)
	}

	for {
		n, err := io.CopyN(r, file, DefaultBlockSize)
		if err != nil {
			break
		}
		task.RSize += n
	}
	m.UpdateTaskStatus(task, FileTaskDone)

	m.logger.Debugf("file: %s, size: %d, status: %s", task.FileName, task.RSize, task.Status)
}

func (m *Manager) UploadFileStream(c *transport.Client, remote string, addr string, tmp *TempFile, ctx context.Context) {

	task := &TaskItem{
		Total:    int64(tmp.Size),
		FileName: tmp.Name,
		Status:   FileTaskRunning,
		Host:     addr,
	}

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

	// 引用计数
	atomic.AddInt32(&tmp.Num, 1)
	defer func() {
		atomic.AddInt32(&tmp.Num, -1)
		if tmp.Num <= 0 {
			os.Remove(tmp.Path)
		}
	}()

	// 打开tmp文件
	file, err := os.OpenFile(tmp.Path, os.O_RDONLY, os.ModePerm)
	if err != nil {
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
		// 遇到关闭连接的情况
		if task.Status != FileTaskDone {
			m.UpdateTaskStatus(task, FileTaskFailed)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			_ = file.Close()
		}
	}()

	readBuf := make([]byte, DefaultBlockSize)
	size := tmp.Size
	for size > 0 {
		n, err := file.Read(readBuf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				//time.Sleep(20 * time.Millisecond)
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
