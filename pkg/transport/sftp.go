package transport

import (
	"github.com/pkg/sftp"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
)

func (c *Client) GetSftpClient() *sftp.Client {
	return c.sftpClient
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
	return c.sftpClient.Mkdir(filepath.ToSlash(dirPath))
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
