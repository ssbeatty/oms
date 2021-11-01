package transport

import (
	"fmt"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"sync"
)

/*
变量声明
*/

var CurrentFiles *sync.Map

const (
	DefaultBlockSize = 1024 * 4
	TaskRunning      = "running"
	TaskDone         = "done"
	TaskFailed       = "failed"
)

func init() {
	CurrentFiles = &sync.Map{}
}

/*
扩展服务
*/

type TaskItem struct {
	Status   string
	Total    int64
	ch       chan int64 // 当前字节的channel
	RSize    int64      // 已传输字节
	CSize    int64      // 当前字节
	FileName string
	Host     string
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

/*
sftp基础服务
*/

func NewClientWithSftp(host string, port int, user string, password string, KeyBytes []byte) (*Client, error) {
	client, err := NewClient(host, port, user, password, KeyBytes)
	if err != nil {
		return nil, err
	}
	err = client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) NewSftpClient() error {
	if c.sftpClient == nil {
		cli, err := sftp.NewClient(c.sshClient)
		if err != nil {
			return err
		}
		c.sftpClient = cli
	}
	return nil
}

func (c *Client) ReadDir(path string) ([]os.FileInfo, error) {
	if c.IsDir(path) {
		info, err := c.sftpClient.ReadDir(path)
		return info, err
	}
	return nil, nil
}

func (c *Client) GetFile(path string) (*sftp.File, error) {
	file, err := c.sftpClient.Open(path)
	if err != nil {
		return nil, err
	}
	return file, err
}

func (c *Client) IsDir(path string) bool {
	// 检查远程是文件还是目录
	info, err := c.sftpClient.Stat(path)
	if err == nil && info.IsDir() {
		return true
	}
	return false
}

func (c *Client) MkdirAll(dirPath string) error {

	parentDir := filepath.ToSlash(filepath.Dir(dirPath))
	_, err := c.sftpClient.Stat(parentDir)
	if err != nil {
		// log.Println(err)
		if err.Error() == "file does not exist" {
			err := c.MkdirAll(parentDir)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	err = c.sftpClient.Mkdir(filepath.ToSlash(dirPath))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Remove(path string) error {
	return c.sftpClient.Remove(path)
}

func (c *Client) RemoveDir(remoteDir string) error {
	remoteFiles, err := c.sftpClient.ReadDir(remoteDir)
	if err != nil {
		return err
	}
	for _, file := range remoteFiles {
		subRemovePath := path.Join(remoteDir, file.Name())
		if file.IsDir() {
			c.RemoveDir(subRemovePath)
		} else {
			c.Remove(subRemovePath)
		}
	}
	c.sftpClient.RemoveDirectory(remoteDir)
	return nil
}

func (c *Client) ReadLink(path string) (string, error) {
	return c.sftpClient.ReadLink(path)
}

func (c *Client) Stat(path string) (os.FileInfo, error) {
	return c.sftpClient.Stat(path)
}

func (c *Client) RealPath(path string) (string, error) {
	return c.sftpClient.RealPath(path)
}

func (c *Client) GetPwd() (string, error) {
	return c.sftpClient.Getwd()
}
