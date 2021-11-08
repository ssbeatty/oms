package ssh

import (
	"fmt"
	"io"
	"mime/multipart"
	"oms/internal/utils"
	"oms/pkg/transport"
)

const (
	DefaultBlockSize = 1024 * 4
	FileTaskRunning  = "running"
	FileTaskDone     = "done"
	FileTaskFailed   = "failed"
)

type TaskItem struct {
	Status   string
	Total    int64      // 文件总字节
	ch       chan int64 // 当前字节的channel
	RSize    int64      // 已传输字节
	CSize    int64      // 当前字节
	FileName string
	Host     string
}

// manageChannel 更新file task pool中file task的传输字节数
func (m *Manager) manageChannel(ch chan int64, key string) {
	for val := range ch {
		item, ok := m.fileList.Load(key)
		if !ok {
			continue
		}
		task := item.(*TaskItem)
		task.RSize += val
	}
}

// UploadFileOneAsync 上传文件并将addr/filename维护到file task pool
func (m *Manager) UploadFileOneAsync(c *transport.Client, fileH *multipart.FileHeader, remote string, addr string, filename string) {
	ch := make(chan int64, 10)
	task := &TaskItem{
		Total:    fileH.Size,
		ch:       ch,
		FileName: fileH.Filename,
		Status:   FileTaskRunning,
		Host:     addr,
	}

	file, err := fileH.Open()
	if err != nil {
		m.logger.Errorf("error when open multipart file, err: %v", err)
		task.Status = FileTaskFailed
		return
	}

	remoteFile, remoteDir := utils.ParseUploadPath(remote, fileH.Filename)
	if _, err := c.GetSftpClient().Stat(remoteDir); err != nil {
		if err := c.MkdirAll(remoteDir); err != nil {
			task.Status = FileTaskFailed
			m.logger.Errorf("error when sftp create dirs, err: %v", err)
		}
	}
	r, err := c.GetSftpClient().Create(remoteFile)
	if err != nil {
		m.logger.Errorf("error when sftp create file, err: %v", err)
		task.Status = FileTaskFailed
		return
	}

	defer func() {
		m.logger.Debugf("upload file goroutine exit.")
		_ = file.Close()
		_ = r.Close()
		close(ch)
		// 遇到关闭连接的情况
		if task.Status != FileTaskDone {
			task.Status = FileTaskFailed
		}
	}()

	key := fmt.Sprintf("%s/%s", addr, filename)
	m.fileList.Store(key, task)

	go m.manageChannel(ch, key)

	for {
		n, err := io.CopyN(r, file, DefaultBlockSize)
		ch <- n
		if err != nil {
			break
		}
	}
	task.Status = FileTaskDone

	m.logger.Debugf("file: %s, size: %d, status: %s", task.FileName, task.RSize, task.Status)
}

// NewClientWithSftp 创建新的ssh client并创建sftp客户端
func (m *Manager) NewClientWithSftp(host string, port int, user string, password string, KeyBytes []byte) (*transport.Client, error) {
	client, err := m.NewClient(host, port, user, password, KeyBytes)
	if err != nil {
		return nil, err
	}
	err = client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}
