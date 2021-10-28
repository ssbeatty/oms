package transport

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"oms/pkg/cache"
	"oms/pkg/utils"
	"path/filepath"
	"sync"
)

var CurrentFiles *sync.Map
var SSHClientPoll *cache.Cache

const (
	DefaultBlockSize = 1024 * 4
	TaskRunning      = "running"
	TaskDone         = "done"
	TaskFailed       = "failed"
)

func init() {
	CurrentFiles = &sync.Map{}
	SSHClientPoll = cache.NewCache(1000)
}

type TaskItem struct {
	Status   string
	Total    int64
	ch       chan int64 // 当前字节的channel
	RSize    int64      // 已传输字节
	CSize    int64      // 当前字节
	FileName string
	Host     string
}

func (h *Config) serialize() int64 {
	return utils.InetAtoN(h.Host, h.Port)
}

func NewClient(host string, port int, user string, password string, KeyBytes []byte) (*Client, error) {
	if user == "" {
		user = "root"
	}
	var config = &Config{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   password,
		Passphrase: password,
	}
	if KeyBytes != nil {
		config.KeyBytes = KeyBytes
	}
	if cli, ok := SSHClientPoll.Get(config.serialize()); ok {
		ss, err := cli.(*Client).NewSession()
		defer ss.Close()

		if err != nil {
			SSHClientPoll.Remove(config.serialize())
		} else {
			return cli.(*Client), nil
		}
	}

	cli, err := New(config)
	if err != nil {
		return nil, err
	}
	SSHClientPoll.Add(config.serialize(), cli)
	return cli, nil
}

func manageChannel(ch chan int64, key string) {
	for val := range ch {
		item, ok := CurrentFiles.Load(key)
		if !ok {
			continue
		}
		task := item.(*TaskItem)
		task.RSize += val
	}
}

func (c *Client) UploadFileOne(fileH *multipart.FileHeader, remote string) error {
	file, err := fileH.Open()
	if err != nil {
		return err
	}
	defer file.Close()
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
	if _, err := c.sftpClient.Stat(remoteDir); err != nil {
		log.Println("sftp: Mkdir all", remoteDir)
		_ = c.MkdirAll(remoteDir)
	}
	r, err := c.sftpClient.Create(remoteFile)
	if err != nil {
		return err
	}

	defer r.Close()

	_, err = io.Copy(r, file)
	return err
}

func (c *Client) UploadFileOneAsync(fileH *multipart.FileHeader, remote string, addr string, filename string) {
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
	if _, err := c.sftpClient.Stat(remoteDir); err != nil {
		log.Println("sftp: Mkdir all", remoteDir)
		if err := c.MkdirAll(remoteDir); err != nil {
			task.Status = TaskFailed
			log.Errorf("error when sftp create dirs, err: %v", err)
		}
	}
	r, err := c.sftpClient.Create(remoteFile)
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
	CurrentFiles.Store(key, task)

	go manageChannel(ch, key)

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
