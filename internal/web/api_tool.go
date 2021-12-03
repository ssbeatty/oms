package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/fs"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"oms/internal/ssh"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func (s *Service) RunCmd(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	pType := c.Query("type")
	cmd := c.Query("cmd")
	if cmd == "" {
		data := generateResponsePayload(HttpStatusError, "cmd can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	sudoRaw := c.Query("sudo")
	// default false
	sudo, _ := strconv.ParseBool(sudoRaw)

	hosts := s.ParseHostList(pType, id)
	if len(hosts) == 0 {
		data := generateResponsePayload(HttpStatusError, "parse host array empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// do cmd
	results := s.RunCmdExec(hosts, cmd, sudo)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)

}

func (s *Service) FileUploadUnBlock(c *gin.Context) {
	var remoteFile string
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	form, _ := c.MultipartForm()
	files := form.File["files"]
	remote := c.PostForm("remote")
	if remote == "" {
		remoteFile = remote
	} else {
		if remote[len(remote)-1] == '/' {
			remoteFile = remote
		} else {
			remoteFile = remote + "/"
		}
	}
	pType := c.PostForm("type")
	hosts := s.ParseHostList(pType, id)

	s.UploadFileUnBlock(hosts, files, remoteFile)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)

}

func (s *Service) GetPathInfo(c *gin.Context) {
	p := c.Query("id")
	id, err := strconv.Atoi(c.Query("host_id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "parse id error", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	results, err := s.GetPathInfoExec(id, p)
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DownLoadFile(c *gin.Context) {
	p := c.Query("id")
	id, err := strconv.Atoi(c.Query("host_id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	file := s.DownloadFile(id, p)

	if file != nil {
		defer file.Close()

		fh, err := file.Stat()
		if err != nil {
			s.logger.Errorf("error when Stat file, err: %v", err)
			c.String(http.StatusOK, "file stat error")
			return
		}
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fh.Name()))

		mode := fh.Mode() & fs.ModeType
		// this is a read able file
		if mode == 0 {
			if fh.Size() != 0 {
				http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), file)
				return
			} else {
				buf, err := ioutil.ReadAll(file)
				if err != nil {
					s.logger.Errorf("read virtual file error: %v", err)
				}
				http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), bytes.NewReader(buf))
				return
			}
		} else {
			c.String(http.StatusOK, fmt.Sprintf("read error, file type is [%s]", mode))
			return
		}
	} else {
		c.String(http.StatusOK, "file not existed")
		return
	}
}

func (s *Service) DeleteFile(c *gin.Context) {
	var data Response
	p := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("host_id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.DeleteFileOrDir(id, p)
	if err != nil {
		data = generateResponsePayload(HttpStatusError, "remove file error", nil)
	} else {
		data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Service) MakeDirRemote(c *gin.Context) {
	var data Response
	p := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("host_id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dirName := c.PostForm("dir")

	err = s.MakeDir(id, p, dirName)
	if err != nil {
		data = generateResponsePayload(HttpStatusError, "mkdir error", nil)
	} else {
		data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Service) ExportData(c *gin.Context) {
	marshal, err := s.ExportDbData()
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "ExportData error when export data", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	file := bytes.NewReader(marshal)
	c.Writer.Header().Set("content-type", "application/json")
	http.ServeContent(c.Writer, c.Request, "export", time.Now(), file)
}

func (s *Service) ImportData(c *gin.Context) {
	fh, err := c.FormFile("dataFile")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			s.logger.Error(err)
			data := generateResponsePayload(HttpStatusError, err.Error(), nil)
			c.JSON(http.StatusOK, data)
			return
		}
		err = s.ImportDbData(fileBytes)
		if err != nil {
			data := generateResponsePayload(HttpStatusError, err.Error(), nil)
			c.JSON(http.StatusOK, data)
			return
		}
	} else {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func (s *Service) FileUploadV2(c *gin.Context) {
	var id int
	var remoteFile, dType string
	files := make(map[string]int)

	// map [filename(url encode)] = size(int)
	fileHeaders := c.GetHeader("X-Files")
	if fileHeaders == "" {
		c.Request.URL.Path += "_file"
		s.engine.HandleContext(c)
		return
	}
	err := json.Unmarshal([]byte(fileHeaders), &files)
	if err != nil || len(files) == 0 {
		c.Request.URL.Path += "_file"
		s.engine.HandleContext(c)
		return
	}

	mediaType, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "parse content type error", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	// 文件必须放到formData的末尾 否则解析会失败
	if strings.HasPrefix(mediaType, "multipart/") {
		partReader := multipart.NewReader(c.Request.Body, params["boundary"])
		for {
			part, err := partReader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if part.FileName() == "" {
				// 如果不是文件放在form里
				all, err := ioutil.ReadAll(part)
				if err != nil {
					s.logger.Errorf("file upload v2 error when read_all form part, err: %v", err)
					continue
				}
				switch part.FormName() {
				case "type":
					dType = string(all)
				case "remote":
					remote := string(all)
					if remote == "" {
						remoteFile = remote
					} else {
						if remote[len(remote)-1] == '/' {
							remoteFile = remote
						} else {
							remoteFile = remote + "/"
						}
					}
				case "id":
					id, err = strconv.Atoi(string(all))
					if err != nil {
						data := generateResponsePayload(HttpStatusError, "parse params id error", nil)
						c.JSON(http.StatusOK, data)
						return
					}
				}
			} else {
				// TODO skip repeat file
				hosts := s.ParseHostList(dType, id)
				// 每一个文件对应一个context如果 文件传输一半终止了 其下面所有的传输终止
				ctx, cancel := context.WithCancel(context.Background())

				fName := part.FileName()
				escape := base64.StdEncoding.EncodeToString([]byte(fName))

				p := path.Join("tmp", fmt.Sprintf("multipart-%d-%s", int(time.Now().Unix()), fName))
				tempFile := ssh.TempFile{
					Name: fName,
					Size: files[escape],
					Path: p,
				}
				tmp, err := os.OpenFile(tempFile.Path, os.O_TRUNC|os.O_RDWR|os.O_CREATE, os.ModePerm)
				if err != nil {
					data := generateResponsePayload(HttpStatusError, "create tmp file error", nil)
					c.JSON(http.StatusOK, data)
					return
				}

				// 在传输每个文件到tmp的同时就开始复制其到sftp客户端
				go s.UploadFileStream(hosts, &tempFile, remoteFile, ctx)

				n, err := io.Copy(tmp, part)
				if err != nil {
					// 这里多半是浏览器取消了请求
					if int(n) != files[escape] {
						cancel() // cancel all

						err = tmp.Close()
						if err != nil {
							s.logger.Errorf("close file %s error, err: %v", tmp.Name(), err)
						}
						os.Remove(tempFile.Path)
					}
					data := generateResponsePayload(HttpStatusError, "io copy error", nil)
					c.JSON(http.StatusOK, data)
					return
				}

				// 传输完成这个句柄是一定要关闭的
				err = tmp.Close()
				if err != nil {
					s.logger.Errorf("close file %s error, err: %v", tmp.Name(), err)
				}
			}
		}
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}
