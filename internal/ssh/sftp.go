package ssh

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"oms/pkg/transport"
	"path/filepath"
)

const (
	DefaultBlockSize = 1024 * 4
	TaskRunning      = "running"
	TaskDone         = "done"
	TaskFailed       = "failed"
)

type TaskItem struct {
	Status   string
	Total    int64
	ch       chan int64 // 当前字节的channel
	RSize    int64      // 已传输字节
	CSize    int64      // 当前字节
	FileName string
	Host     string
}

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

func (m *Manager) UploadFileOneAsync(c *transport.Client, fileH *multipart.FileHeader, remote string, addr string, filename string) {
	ch := make(chan int64, 10)
	task := &TaskItem{
		Total:    fileH.Size,
		ch:       ch,
		FileName: fileH.Filename,
		Status:   TaskRunning,
		Host:     addr,
	}

	file, err := fileH.Open()
	if err != nil {
		log.Errorf("error when open multipart file, err: %v", err)
		task.Status = TaskFailed
		return
	}

	var remoteFile, remoteDir string
	if remote != "" {
		if remote[len(remote)-1] == '/' {
			remoteFile = filepath.ToSlash(filepath.Join(remote, filepath.Base(fileH.Filename)))
			remoteDir = remote
		} else {
			remoteFile = remote
			remoteDir = filepath.ToSlash(filepath.Dir(remoteFile))
		}
	} else {
		remoteFile = fileH.Filename
		remoteDir = filepath.ToSlash(filepath.Dir(remoteFile))
	}
	if _, err := c.GetSftpClient().Stat(remoteDir); err != nil {
		log.Println("sftp: Mkdir all", remoteDir)
		if err := c.MkdirAll(remoteDir); err != nil {
			task.Status = TaskFailed
			log.Errorf("error when sftp create dirs, err: %v", err)
		}
	}
	r, err := c.GetSftpClient().Create(remoteFile)
	if err != nil {
		log.Errorf("error when sftp create file, err: %v", err)
		task.Status = TaskFailed
		return
	}

	defer func() {
		log.Debugf("upload file goroutine exit.")
		_ = file.Close()
		_ = r.Close()
		close(ch)
		// 遇到关闭连接的情况
		if task.Status != TaskDone {
			task.Status = TaskFailed
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
	task.Status = TaskDone

	log.Debugf("file: %s, size: %d, status: %s", task.FileName, task.RSize, task.Status)
}

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
