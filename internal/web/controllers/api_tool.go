package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

const (
	MaxPreviewSize = 8 * 1024 * 1024
)

func (s *Service) RunCmd(c *Context) {
	var params payload.CmdParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		hosts := s.ParseHostList(params.Type, params.Id)
		if len(hosts) == 0 {
			data := payload.GenerateResponsePayload(HttpStatusError, payload.ErrHostParseEmpty, nil)
			c.JSON(http.StatusOK, data)
			c.ResponseError("")
			return
		}
		// do cmd
		results := s.RunCmdExec(hosts, params.Cmd, params.Sudo)

		c.ResponseOk(results)
	}
}

func (s *Service) GetPathInfo(c *Context) {
	var params payload.OptionsFileParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		results, err := s.GetPathInfoExec(params.HostId, params.Id)
		if err != nil {
			data := payload.GenerateResponsePayload(HttpStatusError, err.Error(), nil)
			c.JSON(http.StatusOK, data)
			return
		}
		c.ResponseOk(results)
	}
}

func (s *Service) DownLoadFile(c *Context) {
	var params payload.OptionsFileParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		file := s.DownloadFile(params.HostId, params.Id)
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
			c.Status(http.StatusNotFound)
			return
		}
	}
}

func (s *Service) DeleteFile(c *Context) {
	var params payload.OptionsFileParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err = s.DeleteFileOrDir(params.HostId, params.Id)
		if err != nil {
			c.ResponseError(err.Error())
		} else {
			c.ResponseOk(nil)
		}
	}
}

func (s *Service) MakeDirRemote(c *Context) {
	var params payload.MkdirParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err = s.MakeDir(params.HostId, params.Id, params.Dir)
		if err != nil {
			c.ResponseError(err.Error())
		} else {
			c.ResponseOk(nil)
		}
	}
}

func (s *Service) StartJob(c *Context) {
	var form payload.OptionsJobForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		job, err := models.GetJobById(form.Id)
		if err != nil {
			s.Logger.Errorf("error when get job, err: %v", err)
			c.ResponseError(err.Error())
		}
		err = s.taskManager.StartJob(job)
		if err != nil {
			s.Logger.Errorf("error when start job, err: %v", err)
			c.ResponseError(err.Error())
		}

		_ = models.RefreshJob(job)

		c.ResponseOk(job)
	}
}

func (s *Service) StopJob(c *Context) {
	var form payload.OptionsJobForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		job, err := models.GetJobById(form.Id)
		if err != nil {
			s.Logger.Errorf("error when get job, err: %v", err)
			c.ResponseError(err.Error())
		}
		err = s.taskManager.StopJob(job.Id)
		if err != nil {
			s.Logger.Errorf("error when stop job, err: %v", err)
			c.ResponseError(err.Error())
		}

		_ = models.RefreshJob(job)

		c.ResponseOk(job)
	}
}

func (s *Service) FilePreview(c *Context) {
	var params payload.OptionsFileParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		file := s.DownloadFile(params.HostId, params.Id)
		if file != nil {
			defer file.Close()

			fh, err := file.Stat()
			if err != nil {
				s.Logger.Errorf("error when Stat file, err: %v", err)
				c.ResponseError("file stat error")
			}
			if fh.Size() > MaxPreviewSize {
				c.ResponseError("file size too large")
				return
			}

			mode := fh.Mode() & fs.ModeType
			// this is a read able file
			if mode == 0 {
				buf, err := ioutil.ReadAll(file)
				if err != nil {
					s.Logger.Errorf("read virtual file error: %v", err)
				}
				baseRaw := base64.StdEncoding.EncodeToString(buf)

				c.ResponseOk(baseRaw)
			} else {
				c.ResponseError(fmt.Sprintf("read error, file type is [%s]", mode))
			}
		} else {
			c.ResponseError("file not found")
		}
	}
}

// 这些方法暂时不重构

func (s *Service) FileUploadUnBlock(c *Context) {
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

func (s *Service) FileUploadV2(c *Context) {
	var id int
	var remoteFile, dType string
	files := make(map[string]int)

	// map [filename(url encode)] = size(int)
	fileHeaders := c.GetHeader("X-Files")
	if fileHeaders == "" {
		c.Request.URL.Path += "_file"
		s.Engine.HandleContext(c.Context)
		return
	}
	err := json.Unmarshal([]byte(fileHeaders), &files)
	if err != nil || len(files) == 0 {
		c.Request.URL.Path += "_file"
		s.Engine.HandleContext(c.Context)
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

	_, _ = fmt.Fprintf(w, "[job]: %s, [cmd]: %s log\n", job.Name(), job.Cmd())
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
