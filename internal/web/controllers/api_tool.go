package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hpcloud/tail"
	"io"
	"io/fs"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/web/payload"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func (s *Service) RunCmd(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	pType := c.Query("type")
	cmd := c.Query("cmd")
	if cmd == "" {
		data := payload.GenerateResponsePayload(HttpStatusError, "cmd can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	sudoRaw := c.Query("sudo")
	// default false
	sudo, _ := strconv.ParseBool(sudoRaw)

	hosts := s.ParseHostList(pType, id)
	if len(hosts) == 0 {
		data := payload.GenerateResponsePayload(HttpStatusError, "parse host array empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// do cmd
	results := s.RunCmdExec(hosts, cmd, sudo)
	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)

}

func (s *Service) FileUploadUnBlock(c *gin.Context) {
	var remoteFile string
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
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
	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)

}

func (s *Service) GetPathInfo(c *gin.Context) {
	p := c.Query("id")
	id, err := strconv.Atoi(c.Query("host_id"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "parse id error", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	results, err := s.GetPathInfoExec(id, p)
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DownLoadFile(c *gin.Context) {
	p := c.Query("id")
	id, err := strconv.Atoi(c.Query("host_id"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	file := s.DownloadFile(id, p)

	if file != nil {
		defer file.Close()

		fh, err := file.Stat()
		if err != nil {
			s.Logger.Errorf("error when Stat file, err: %v", err)
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
					s.Logger.Errorf("read virtual file error: %v", err)
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
	var data payload.Response
	p := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("host_id"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.DeleteFileOrDir(id, p)
	if err != nil {
		data = payload.GenerateResponsePayload(HttpStatusError, "remove file error", nil)
	} else {
		data = payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Service) MakeDirRemote(c *gin.Context) {
	var data payload.Response
	p := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("host_id"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dirName := c.PostForm("dir")

	err = s.MakeDir(id, p, dirName)
	if err != nil {
		data = payload.GenerateResponsePayload(HttpStatusError, "mkdir error", nil)
	} else {
		data = payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	}
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
		s.Engine.HandleContext(c)
		return
	}
	err := json.Unmarshal([]byte(fileHeaders), &files)
	if err != nil || len(files) == 0 {
		c.Request.URL.Path += "_file"
		s.Engine.HandleContext(c)
		return
	}

	mediaType, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "parse content type error", nil)
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
					s.Logger.Errorf("file upload v2 error when read_all form part, err: %v", err)
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
						data := payload.GenerateResponsePayload(HttpStatusError, "parse params id error", nil)
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
					data := payload.GenerateResponsePayload(HttpStatusError, "create tmp file error", nil)
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
							s.Logger.Errorf("close file %s error, err: %v", tmp.Name(), err)
						}
						os.Remove(tempFile.Path)
					}
					data := payload.GenerateResponsePayload(HttpStatusError, "io copy error", nil)
					c.JSON(http.StatusOK, data)
					return
				}

				// 传输完成这个句柄是一定要关闭的
				err = tmp.Close()
				if err != nil {
					s.Logger.Errorf("close file %s error, err: %v", tmp.Name(), err)
				}
			}
		}
	}
	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func (s *Service) StartJob(c *Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.Logger.Errorf("StartJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		s.Logger.Errorf("StartJob error when GetJobById, err: %v", err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.StartJob(job)
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	_ = models.RefreshJob(job)

	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) StopJob(c *Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.Logger.Errorf("StopJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		s.Logger.Errorf("StartJob error when GetJobById, err: %v", err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.StopJob(job.Id)
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	_ = models.RefreshJob(job)

	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) RestartJob(c *Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.Logger.Errorf("RestartJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		s.Logger.Errorf("RestartJob error when GetJobById, err: %v", err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	_ = s.taskManager.StopJob(id)

	err = s.taskManager.StartJob(job)
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	_ = models.RefreshJob(job)

	data := payload.GenerateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetLogStream(c *Context) {
	var offset int64
	idRaw := c.Query("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.Logger.Errorf("GetLogStream error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	job, ok := s.taskManager.GetJob(id)
	if !ok {
		c.String(http.StatusOK, "job is stopped")
		return
	}

	w := c.Writer
	header := w.Header()
	//http chunked
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Content-Type", "text/plain;charset=utf-8")
	// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/X-Content-Type-Options
	// 取消浏览器的MIME嗅探算法
	header.Set("X-Content-Type-Options", "nosniff")

	_, _ = fmt.Fprintln(w, fmt.Sprintf("[job]: %s, [cmd]: %s log", job.Name(), job.Cmd()))
	w.(http.Flusher).Flush()

	stat, err := os.Stat(job.Log())
	if err != nil {
		c.String(http.StatusOK, "log file not existed")
		return
	}
	if stat.Size() > 2000 {
		offset = -2000
	} else {
		offset = -stat.Size()
	}

	t, err := tail.TailFile(job.Log(), tail.Config{
		Follow:   true,
		Poll:     true,
		Location: &tail.SeekInfo{Offset: offset, Whence: io.SeekEnd},
	})
	if err != nil {
		data := payload.GenerateResponsePayload(HttpStatusError, "tail file error", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	defer func() {
		err := t.Stop()
		if err != nil {
			s.Logger.Errorf("error when stop tail, err: %v", err)
			return
		}
		s.Logger.Debug("GetLogStream log file stream exit.")
	}()

	for {
		select {
		case line := <-t.Lines:
			if line == nil {
				continue
			}
			_, err := fmt.Fprintln(w, line.Text)
			if err != nil {
				return
			}
			w.(http.Flusher).Flush()
		case <-w.CloseNotify():
			s.Logger.Debug("log stream got done notify.")
			return
		}
	}
}