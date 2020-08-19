package ssh

import (
	"io"
	"mime/multipart"
	"oms/logger"
	"path/filepath"
)

func (c *Client) UploadFileOne(fileH *multipart.FileHeader, remote string) error {
	file, err := fileH.Open()
	if err != nil {
		logger.Logger.Println(err)
	}
	var remoteFile, remoteDir string
	if remote[len(remote)-1] == '/' {
		remoteFile = filepath.ToSlash(filepath.Join(remote, filepath.Base(fileH.Filename)))
		remoteDir = remote
	} else {
		remoteFile = remote
		remoteDir = filepath.ToSlash(filepath.Dir(remoteFile))
	}
	if fileH.Size > 1000 {
		rsum := c.Md5File(remoteFile)
		if rsum != "" {
			lsum, _ := Md5File2(file)
			if lsum == rsum {
				logger.Logger.Println("sftp: 文件与本地一致，跳过上传！", fileH.Filename)
				return nil
			}
			logger.Logger.Println("sftp: 正在上传 ", fileH.Filename)
		}
	}
	if _, err := c.SFTPClient.Stat(remoteDir); err != nil {
		logger.Logger.Println("sftp: Mkdir all", remoteDir)
		c.MkdirAll(remoteDir)
	}
	r, err := c.SFTPClient.Create(remoteFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(r, file)
	return err
}
