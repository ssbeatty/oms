package transport

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
)

const (
	ShellTmpPath    = ".oms"
	WindowsShellExt = ".bat"
	LinuxShellExt   = ".sh"
)

func (c *Client) GetSftpClient() *sftp.Client {
	return c.sftpClient
}

func (c *Client) UploadFile(local, fPath string, fName string) error {
	if c.sftpClient == nil {
		err := c.NewSftpClient()
		if err != nil {
			return err
		}
	}
	r, err := os.OpenFile(local, os.O_RDONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	fPath = filepath.ToSlash(fPath)

	if ok := c.PathExists(fPath); !ok {
		_ = c.MkdirAll(filepath.Dir(fPath))
	}

	if fh, err := c.sftpClient.Stat(fPath); err == nil && fh.IsDir() {
		fPath = filepath.ToSlash(filepath.Join(fPath, fName))
	}

	w, err := c.sftpClient.Create(fPath)
	if err != nil {
		return err
	}

	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) UploadHttpFile(fileH *multipart.FileHeader, remote string) error {
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

func (c *Client) UploadFileRaw(context string, fPath string) error {
	if c.sftpClient == nil {
		err := c.NewSftpClient()
		if err != nil {
			return err
		}
	}
	fPath = filepath.ToSlash(fPath)

	if _, err := c.sftpClient.Stat(fPath); err != nil {
		_ = c.MkdirAll(filepath.Dir(fPath))
	}

	r, err := c.sftpClient.Create(fPath)
	if err != nil {
		return err
	}

	defer r.Close()

	_, err = r.Write([]byte(context))
	if err != nil {
		return err
	}
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
	return c.sftpClient.MkdirAll(filepath.ToSlash(dirPath))
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

func (c *Client) PathExists(path string) bool {
	_, err := c.sftpClient.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func (c *Client) RealPath(path string) (string, error) {
	return c.sftpClient.RealPath(path)
}

func (c *Client) GetPwd() (string, error) {
	return c.sftpClient.Getwd()
}

func (s *Session) RunScript(shell string, sudo bool) ([]byte, error) {
	fPath := filepath.ToSlash(filepath.Join(ShellTmpPath, uuid.NewString()))

	if s.Client.GetTargetMachineOs() != GOOSWindows {
		fPath += LinuxShellExt
	} else {
		fPath += WindowsShellExt
	}

	err := s.Client.UploadFileRaw(shell, fPath)
	if err != nil {
		return nil, err
	}

	defer s.Client.Remove(fPath)

	command := fPath

	if s.Client.GetTargetMachineOs() != GOOSWindows {
		command = fmt.Sprintf("chmod +x %s;%s", fPath, fPath)
	}
	if sudo {
		return s.Sudo(command, s.Client.Conf.Password)
	} else {
		return s.Output(command)
	}
}

func (c *Client) Chmod(path string) error {
	fh, err := c.sftpClient.Stat(path)
	if err != nil {
		return err
	}
	if fh.IsDir() {
		walker := c.sftpClient.Walk(path)
		for walker.Step() {
			if walker.Err() != nil {
				continue
			}
			err := c.sftpClient.Chmod(walker.Path(), fs.ModePerm)
			if err != nil {
				return err
			}
		}
	} else {
		err := c.sftpClient.Chmod(path, fs.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}
