package controllers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/gocarina/gocsv"
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/models"
	"github.com/ssbeatty/oms/internal/ssh"
	"github.com/ssbeatty/oms/internal/task"
	"github.com/ssbeatty/oms/internal/web/payload"
	"github.com/ssbeatty/oms/pkg/utils"
	"io"
	"io/fs"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	MaxPreviewSize = 8 * 1024 * 1024
	Utf8Dom        = "\xEF\xBB\xBF"
)

var (
	red   = color.New(color.FgRed).SprintFunc()
	blue  = color.New(color.FgBlue).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

// @BasePath /api/v1

// RunCmd
// @Summary 执行一条命令
// @Description 执行一条命令
// @Param id query integer true "执行者 ID"
// @Param type query string true "执行者类型" example(host,group,tag)
// @Param cmd query string true "命令"
// @Param sudo query bool false "是否sudo执行"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]ssh.Result}
// @Failure 400 {object} payload.Response
// @Router /tools/cmd [get]
func (s *Service) RunCmd(c *Context) {
	var params payload.CmdParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		hosts, err := models.ParseHostList(params.Type, params.Id)
		if err != nil || len(hosts) == 0 {
			data := payload.GenerateErrorResponse(HttpStatusError, payload.ErrHostParseEmpty)
			c.JSON(http.StatusOK, data)
			c.ResponseError("")
			return
		}
		// do cmd
		results := s.RunCmdExec(hosts, params.Cmd, params.Sudo)

		c.ResponseOk(results)
	}
}

// GetPathInfo
// @Summary 目录列表
// @Description 目录列表
// @Param id query string true "远端文件路径"
// @Param host_id query integer true "主机 ID"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=FilePath}
// @Failure 400 {object} payload.Response
// @Router /tools/browse [get]
func (s *Service) GetPathInfo(c *Context) {
	var params payload.OptionsFileParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		results, err := s.GetPathInfoExec(params.HostId, params.Id)
		if err != nil {
			data := payload.GenerateErrorResponse(HttpStatusError, err.Error())
			c.JSON(http.StatusOK, data)
			return
		}
		c.ResponseOk(results)
	}
}

// DownLoadFile
// @Summary 下载文件
// @Description 下载文件
// @Param id query string true "远端文件路径"
// @Param host_id query integer true "主机 ID"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce application/octet-stream
// @Success 200
// @Failure 400 {object} payload.Response
// @Router /tools/download [get]
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

// DeleteFile
// @Summary 删除文件
// @Description 删除文件
// @Param id formData string true "远端文件路径"
// @Param host_id formData integer true "主机 ID"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tools/delete [post]
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

// MakeDirRemote
// @Summary 创建文件夹
// @Description 创建文件夹
// @Param id formData string true "远端文件路径"
// @Param host_id formData integer true "主机 ID"
// @Param dir formData string true "远端目录地址"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tools/mkdir [post]
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

// ExecJob
// @Summary 单次执行任务
// @Description 单次执行任务
// @Param id formData integer true "任务 ID"
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Job}
// @Failure 400 {object} payload.Response
// @Router /job/exec [post]
func (s *Service) ExecJob(c *Context) {
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
		err = s.taskManager.ExecJob(job)
		if err != nil {
			s.Logger.Errorf("error when start job, err: %v", err)
			c.ResponseError(err.Error())
		}

		c.ResponseOk(job)
	}
}

// StartJob
// @Summary 启动任务调度
// @Description 启动任务调度
// @Param id formData integer true "任务 ID"
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Job}
// @Failure 400 {object} payload.Response
// @Router /job/start [post]
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

// StopJob
// @Summary 停止任务调度
// @Description 停止任务调度
// @Param id formData integer true "任务 ID"
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Job}
// @Failure 400 {object} payload.Response
// @Router /job/stop [post]
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

// FilePreview
// @Summary 文件预览
// @Description 文件预览
// @Param id query string true "远端文件路径"
// @Param host_id query integer true "主机 ID"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tools/preview [get]
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

// ModifyFile
//	@Summary		文件修改
//	@Description	文件修改
//	@Param			id		query	string	true	"远端文件路径"
//	@Param			host_id	query	integer	true	"主机 ID"
//	@Tags			tool
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Success		200	{object}	payload.Response
//	@Failure		400	{object}	payload.Response
//	@Router			/tools/modify [post]
func (s *Service) ModifyFile(c *Context) {
	var params payload.ModifyFileParams
	err := c.ShouldBind(&params)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		file := s.GetRWFile(params.HostId, params.Id)
		if file != nil {
			defer file.Close()
			ModifyContentByte, err := base64.StdEncoding.DecodeString(params.ModifyContent)
			if err != nil {
				c.ResponseError(fmt.Sprintf("修改内容解码失败%v", err))
				return
			}
			_, err = file.Write(ModifyContentByte)
			if err == nil {
				c.ResponseOk("文件修改成功")
			} else {
				c.ResponseError(fmt.Sprintf("文件修改失败%v", err))
			}
		}
	}
}

// FileUploadV2
// @Summary 上传文件到主机
// @Description 上传文件到主机
// @Param X-Files header string true "预处理文件列表" example({"filename base64": "file.size"})
// @Param id formData integer true "执行者 ID"
// @Param type formData string true "执行者类型" example(host,group,tag)
// @Param remote formData string true "远端文件夹"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tools/upload [post]
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
		data := payload.GenerateErrorResponse(HttpStatusError, "parse content type error")
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
						data := payload.GenerateErrorResponse(HttpStatusError, "parse params id error")
						c.JSON(http.StatusOK, data)
						return
					}
				}
			} else {
				// TODO skip repeat file
				hosts, err := models.ParseHostList(dType, id)
				if err != nil || len(hosts) == 0 {
					data := payload.GenerateErrorResponse(HttpStatusError, "hosts parse error")
					c.JSON(http.StatusOK, data)
					return
				}
				// 每一个文件对应一个context如果 文件传输一半终止了 其下面所有的传输终止
				ctx, cancel := context.WithCancel(context.Background())

				fName := part.FileName()
				escape := base64.StdEncoding.EncodeToString([]byte(fName))

				p := path.Join(path.Join(s.conf.DataPath, config.DefaultTmpPath), fmt.Sprintf("multipart-%d-%s", int(time.Now().Unix()), fName))
				tempFile := ssh.TempFile{
					Name: fName,
					Size: files[escape],
					Path: p,
				}
				tmp, err := os.OpenFile(tempFile.Path, os.O_TRUNC|os.O_RDWR|os.O_CREATE, os.ModePerm)
				if err != nil {
					data := payload.GenerateErrorResponse(HttpStatusError, "create tmp file error")
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
					data := payload.GenerateErrorResponse(HttpStatusError, "io copy error")
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
	data := payload.GenerateMsgResponse(HttpStatusOk, HttpResponseSuccess)
	c.JSON(http.StatusOK, data)
}

// FileUploadCancel
// @Summary 取消文件上传任务
// @Description 取消文件上传任务
// @Param addr formData string true "地址和端口"
// @Param file formData string true "文件名"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tools/upload/cancel [post]
func (s *Service) FileUploadCancel(c *Context) {
	var form payload.FileTaskCancelForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		key := fmt.Sprintf("%s/%s", form.Addr, form.File)
		s.sshManager.CancelTask(key)

		c.ResponseOk(nil)
	}
}

// deprecated
//func (s *Service) GetLogStream(c *Context) {
//	var offset int64
//	idRaw := c.Query("id")
//	id, err := strconv.Atoi(idRaw)
//	if err != nil {
//		s.Logger.Errorf("GetLogStream error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
//		data := payload.GenerateResponsePayload(HttpStatusError, "can not parse param id", nil)
//		c.JSON(http.StatusOK, data)
//		return
//	}
//
//	job, ok := s.taskManager.GetJob(id)
//	if !ok {
//		c.String(http.StatusOK, "job is stopped")
//		return
//	}
//
//	w := c.Writer
//	header := w.Header()
//	//http chunked
//	header.Set("Transfer-Encoding", "chunked")
//	header.Set("Content-Type", "text/plain;charset=utf-8")
//	// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/X-Content-Type-Options
//	// 取消浏览器的MIME嗅探算法
//	header.Set("X-Content-Type-Options", "nosniff")
//
//	_, _ = fmt.Fprintf(w, "[job]: %s, [cmd]: %s log\n", job.Name(), job.Cmd())
//	w.(http.Flusher).Flush()
//
//	// todo
//	stat, err := os.Stat("")
//	if err != nil {
//		c.String(http.StatusOK, "log file not existed")
//		return
//	}
//	if stat.Size() > 2000 {
//		offset = -2000
//	} else {
//		offset = -stat.Size()
//	}
//
//	t, err := tail.TailFile("", tail.Config{
//		Follow:   true,
//		Poll:     true,
//		Location: &tail.SeekInfo{Offset: offset, Whence: io.SeekEnd},
//	})
//	if err != nil {
//		data := payload.GenerateResponsePayload(HttpStatusError, "tail file error", nil)
//		c.JSON(http.StatusOK, data)
//		return
//	}
//
//	defer func() {
//		err := t.Stop()
//		if err != nil {
//			s.Logger.Errorf("error when stop tail, err: %v", err)
//			return
//		}
//		s.Logger.Debug("GetLogStream log file stream exit.")
//	}()
//
//	for {
//		select {
//		case line := <-t.Lines:
//			if line == nil {
//				continue
//			}
//			_, err := fmt.Fprintln(w, line.Text)
//			if err != nil {
//				return
//			}
//			w.(http.Flusher).Flush()
//		case <-w.CloseNotify():
//			s.Logger.Debug("log stream got done notify.")
//			return
//		}
//	}
//}

func (s *Service) DownloadInstanceLog(c *Context) {
	var param payload.GetTaskInstanceLogParam
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var instance *models.TaskInstance
		instance, err = models.GetTaskInstanceById(param.Id)
		if err != nil {
			s.Logger.Errorf("get instance error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		file, err := os.OpenFile(instance.LogPath, os.O_RDONLY, fs.ModePerm)
		if file != nil && err == nil {
			defer file.Close()

			fh, err := file.Stat()
			if err != nil {
				s.Logger.Errorf("error when Stat file, err: %v", err)
				c.String(http.StatusOK, "file stat error")
				return
			}
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fh.Name()))

			// this is a read able file
			http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), file)
			return
		} else {
			c.Status(http.StatusNotFound)
			return
		}
	}
}

// GetInstanceLog
// @Summary 获取任务执行日志
// @Description 获取任务执行日志
// @Param id query integer true "执行记录 ID"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /task/instance/log/get [get]
func (s *Service) GetInstanceLog(c *Context) {
	var (
		param payload.GetTaskInstanceLogParam
	)

	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var instance *models.TaskInstance
		instance, err = models.GetTaskInstanceById(param.Id)
		if err != nil {
			s.Logger.Errorf("get instance error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		file, err := os.OpenFile(instance.LogPath, os.O_RDONLY, fs.ModePerm)
		if file != nil && err == nil {
			defer file.Close()

			var (
				buffer         bytes.Buffer
				host           *models.Host
				idx            int
				total, success int
			)
			buffer.WriteString(blue(strings.Repeat("#", 40)) + "\r\n\r\n")
			buffer.WriteString(green("#  start run  #\r\n"))
			buffer.WriteString(fmt.Sprintf("Id    : %s\r\n", blue(instance.Id)))
			buffer.WriteString(fmt.Sprintf("Job   : %s\r\n", blue(instance.Job.Name)))
			if instance.Job.CmdType == ssh.CMDTypeShell {
				buffer.WriteString(fmt.Sprintf("Cmd   : %s\r\n", blue(instance.Job.Cmd)))
			} else {
				buffer.WriteString(fmt.Sprintf("Player: %s\r\n", blue(instance.Job.CmdId)))
			}
			buffer.WriteString(fmt.Sprintf("Start : %s\r\n", blue(instance.StartTime.Format(time.RFC3339))))
			buffer.WriteString(fmt.Sprintf("End   : %s\r\n", blue(instance.EndTime.Format(time.RFC3339))))
			buffer.WriteString(fmt.Sprintf("Usage : %s\r\n", blue(instance.EndTime.Sub(instance.StartTime))))

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()

				if strings.HasPrefix(line, task.MarkText) {
					idx++
					total++

					idRaw := regexp.MustCompile("\\d+").FindString(line)
					hostId, err := strconv.Atoi(idRaw)
					if err != nil {
						s.Logger.Errorf("error when parse host_id from log, instance_id: %d, err: %v", instance.Id, err)
						continue
					}

					host, err = models.GetHostById(hostId)
					if err != nil {
						continue
					}
					if strings.HasSuffix(line, task.ErrorText) {
						buffer.WriteString(red(fmt.Sprintf("## Seq: %d host info ##\r\n", idx)))
					} else {
						success++
						buffer.WriteString(green(fmt.Sprintf("## Seq: %d host info ##\r\n", idx)))
					}
					buffer.WriteString(fmt.Sprintf("Host: %s\tId: %s\r\n", blue(host.Name), blue(host.Id)))
					buffer.WriteString(fmt.Sprintf("Addr: %s\r\n", blue(fmt.Sprintf("%s:%d", host.Addr, host.Port))))
					buffer.WriteString(strings.Repeat("-", 40) + "\r\n")
				} else if strings.HasPrefix(line, task.DoneMartText) {
					buffer.WriteString("\r\n")
					buffer.WriteString(blue(fmt.Sprintf("执行完毕, 一共: %d个主机, 成功: %d个\r\n", total, success)))
				} else {
					buffer.WriteString(line)
					buffer.WriteString("\r\n")
				}
			}
			buffer.WriteString(blue(strings.Repeat("#", 40)) + "\r\n")

			if err := scanner.Err(); err != nil {
				s.Logger.Errorf("error when scanner log file, err: %v", err)
			}

			c.ResponseOk(buffer.String())
		} else {
			c.ResponseError("can not found logs")
			return
		}
	}
}

// DataExport
// @Summary 导出资产文件csv
// @Description 导出资产文件csv
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce application/octet-stream
// @Success 200
// @Failure 400 {object} payload.Response
// @Router /tools/export [get]
func (s *Service) DataExport(c *Context) {
	var (
		b    bytes.Buffer
		data []models.HostExport
	)

	defer b.Reset()

	hosts, err := models.GetAllHost()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	for _, host := range hosts {
		tags := make([]string, 0)
		for _, t := range host.Tags {
			tags = append(tags, t.Name)
		}

		data = append(data, models.HostExport{
			Name:        host.Name,
			User:        host.User,
			Addr:        host.Addr,
			PassWord:    host.PassWord,
			Port:        host.Port,
			VNCPort:     host.VNCPort,
			Group:       host.Group.Name,
			GroupParams: host.Group.Params,
			Tags:        tags,
			KeyFile:     host.PrivateKey.KeyFile,
			KeyName:     host.PrivateKey.Name,
			KeyPhrase:   host.PrivateKey.Passphrase,
		})
	}

	// 写入UTF-8 BOM
	b.WriteString(Utf8Dom)
	err = gocsv.Marshal(&data, &b)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.Writer.Header().Set("Content-type", "application/octet-stream")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s", "export.csv"))
	c.String(http.StatusOK, b.String())
}

// DataImport
// @Summary 导入资产文件csv
// @Description 导入资产文件csv
// @Param files formData file true "csv文件"
// @Tags tool
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.HostExport}
// @Failure 400 {object} payload.Response
// @Router /tools/import [post]
func (s *Service) DataImport(c *Context) {
	const (
		EmptyFileError = "empty files"
	)

	var (
		data []models.HostExport
		resp payload.ImportResponse
	)

	form, err := c.MultipartForm()
	if err != nil {
		c.ResponseError(EmptyFileError)
		return
	}
	files := form.File["files"]

	if len(files) == 0 {
		c.ResponseError(EmptyFileError)
		return
	}

	// 多个文件只取第一个
	csvFile := files[0]

	fn, err := csvFile.Open()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	content, err := ioutil.ReadAll(fn)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	// issue https://github.com/gocarina/gocsv/issues/191
	content = bytes.TrimPrefix(content, []byte(Utf8Dom))

	if !utils.IsUtf8(content) {
		content, err = utils.GbkToUtf8(content)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
	}

	err = gocsv.UnmarshalBytes(content, &data)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	for _, row := range data {
		var (
			groupId, privateKeyID int
			tags                  []int
		)

		// has grouped
		if row.Group != "" {
			var (
				err   error
				group *models.Group
			)

			if !models.ExistedGroup(row.Group) {
				if row.GroupParams == "" {
					group, err = models.InsertGroup(row.Group, "", models.GroupHostMode)
				} else {
					group, err = models.InsertGroup(row.Group, row.GroupParams, models.GroupOtherMode)
				}
				if err == nil {
					resp.CreateGroup = append(resp.CreateGroup, group.Name)
				}
			} else {
				group, _ = models.GetGroupByName(row.Group)
			}

			groupId = group.Id
		}

		for _, t := range row.Tags {
			var (
				tag *models.Tag
			)

			if !models.ExistedTag(t) {
				tag, err = models.InsertTag(t)
				if err == nil {
					resp.CreateTag = append(resp.CreateTag, tag.Name)
				}
			} else {
				tag, _ = models.GetTagByName(t)
			}
			tags = append(tags, tag.Id)
		}

		if row.KeyFile != "" {
			var (
				privateKey *models.PrivateKey
			)
			if !models.ExistedPrivateKey(row.KeyFile) {
				privateKey, err = models.InsertPrivateKey(row.KeyName, row.KeyFile, row.KeyPhrase)
				if err == nil {
					resp.CreatePrivateKey = append(resp.CreatePrivateKey, privateKey.Name)
				}
			} else {
				privateKey, err = models.GetPrivateKeyByName(row.KeyName)
			}
			privateKeyID = privateKey.Id
		}

		if !models.ExistedHost(row.Name, row.Addr) {
			h, err := models.InsertHost(
				row.Name, row.User, row.Addr, row.Port, row.PassWord, groupId, tags, privateKeyID, row.VNCPort,
			)
			if err == nil {
				resp.CreateHost = append(resp.CreateHost, h.Name)
			}
		}
	}

	c.ResponseOk(resp)
}
