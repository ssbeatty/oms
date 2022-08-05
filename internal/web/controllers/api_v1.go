package controllers

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"io"
	"io/fs"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/task"
	"oms/internal/web/payload"
	"os"
	"path/filepath"
	"time"
)

const (
	HttpStatusOk        = "200"
	HttpStatusError     = "400"
	HttpResponseSuccess = "success"
)

var parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// api for table host

func (s *Service) GetHosts(c *Context) {
	var (
		hosts []*models.Host
		param payload.GetAllHostParam
	)
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		_, err := models.GetPaginateQuery[*[]*models.Host](
			&hosts, param.PageSize, param.PageNum, nil, true)
		if err != nil {
			s.Logger.Errorf("get all host error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(hosts)
	}
}

func (s *Service) GetOneHost(c *Context) {
	var param payload.GetHostParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		host, err := models.GetHostByIdWithPreload(param.Id)
		if err != nil {
			s.Logger.Errorf("get one host error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(host)
	}
}

func (s *Service) PostHost(c *Context) {
	var form payload.PostHostForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var tags []int
		_ = json.Unmarshal([]byte(form.Tags), &tags)
		host, err := models.InsertHost(form.HostName, form.User, form.Addr, form.Port, form.PassWord, form.Group, tags, form.PrivateKeyId, form.VNCPort)
		if err != nil {
			s.Logger.Errorf("insert host error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(host)
	}
}

func (s *Service) PutHost(c *Context) {
	var form payload.PutHostForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var tags []int
		_ = json.Unmarshal([]byte(form.Tags), &tags)
		host, err := models.UpdateHost(form.Id, form.HostName, form.User, form.Addr, form.Port, form.PassWord, form.Group, tags, form.PrivateKeyId, form.VNCPort)
		if err != nil {
			s.Logger.Errorf("update host error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		// 清理缓存的client
		defer s.sshManager.RemoveCache(host)
		c.ResponseOk(host)
	}
}

func (s *Service) DeleteHost(c *Context) {
	var param payload.DeleteHostParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		_, err := models.DeleteHostById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete host error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}

// api for table PrivateKey

func (s *Service) GetPrivateKeys(c *Context) {
	privateKeys, err := models.GetAllPrivateKey()
	if err != nil {
		s.Logger.Errorf("get all privateKey error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(privateKeys)
}

func (s *Service) GetOnePrivateKey(c *Context) {
	var param payload.GetPrivateKeyParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		privateKey, err := models.GetPrivateKeyById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one privateKey error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(privateKey)
	}
}

func (s *Service) PostPrivateKey(c *Context) {
	var form payload.PostPrivateKeyForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		fileBytes, err := c.readFormFile(form.KeyFile)
		if err != nil {
			s.Logger.Errorf("read form key_file error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		privateKey, err := models.InsertPrivateKey(form.Name, string(fileBytes), form.Passphrase)
		if err != nil {
			s.Logger.Errorf("insert privateKey error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(privateKey)
	}
}

func (s *Service) PutPrivateKey(c *Context) {
	var form payload.PutPrivateKeyForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var fileBytes []byte
		if form.KeyFile != nil {
			fileBytes, err = c.readFormFile(form.KeyFile)
			if err != nil {
				s.Logger.Errorf("read form key_file error: %v", err)
				c.ResponseError(err.Error())
				return
			}
		}
		privateKey, err := models.UpdatePrivateKey(form.Id, form.Name, string(fileBytes), form.Passphrase)
		if err != nil {
			s.Logger.Errorf("update privateKey error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(privateKey)
	}
}

func (s *Service) DeletePrivateKey(c *Context) {
	var param payload.DeletePrivateKeyParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeletePrivateKeyById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete privateKey error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}

// api for table group

func (s *Service) GetGroups(c *Context) {
	groups, err := models.GetAllGroup()
	if err != nil {
		s.Logger.Errorf("get all group error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(groups)
}

func (s *Service) GetOneGroup(c *Context) {
	var param payload.GetGroupParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		group, err := models.GetGroupById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one group error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(group)
	}
}

func (s *Service) PostGroup(c *Context) {
	var form payload.PostGroupForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		group, err := models.InsertGroup(form.Name, form.Params, form.Mode)
		if err != nil {
			s.Logger.Errorf("insert group error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(group)
	}
}

func (s *Service) PutGroup(c *Context) {
	var form payload.PutGroupForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		group, err := models.UpdateGroup(form.Id, form.Name, form.Params, form.Mode)
		if err != nil {
			s.Logger.Errorf("update group error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(group)
	}
}

func (s *Service) DeleteGroup(c *Context) {
	var param payload.DeleteGroupParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeleteGroupById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete group error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}

// api for table tag

func (s *Service) GetTags(c *Context) {
	tags, err := models.GetAllTag()
	if err != nil {
		s.Logger.Errorf("get all tag error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(tags)
}

func (s *Service) GetOneTag(c *Context) {
	var param payload.GetTagParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		tag, err := models.GetTagById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one tag error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(tag)
	}
}

func (s *Service) PostTag(c *Context) {
	var form payload.PostTagForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		tag, err := models.InsertTag(form.Name)
		if err != nil {
			s.Logger.Errorf("insert tag error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(tag)
	}
}

func (s *Service) PutTag(c *Context) {
	var form payload.PutTagForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		tag, err := models.UpdateTag(form.Id, form.Name)
		if err != nil {
			s.Logger.Errorf("update tag error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(tag)
	}
}

func (s *Service) DeleteTag(c *Context) {
	var param payload.DeleteTagParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeleteTagById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete tag error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}

// api for table tunnel

func (s *Service) GetTunnels(c *Context) {
	var param payload.GetTunnelsParam
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var tunnels []*models.Tunnel
		if param.HostId > 0 {
			tunnels, err = models.GetTunnelsByHostId(param.HostId)
		} else {
			tunnels, err = models.GetAllTunnel()
		}
		if err != nil {
			s.Logger.Errorf("get all tunnel error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(tunnels)
	}
}

func (s *Service) GetOneTunnel(c *Context) {
	var param payload.GetTunnelParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		tunnel, err := models.GetTunnelById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one tunnel error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(tunnel)
	}
}

func (s *Service) PostTunnel(c *Context) {
	var form payload.PostTunnelForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		host, err := models.GetHostById(form.HostId)
		if err != nil {
			s.Logger.Errorf("create tunnel error when get host: %v", err)
			c.ResponseError(err.Error())
			return
		}
		tunnel, err := models.InsertTunnel(form.Mode, form.Source, form.Destination, host)
		if err != nil {
			s.Logger.Errorf("insert tunnel error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		err = s.tunnelManager.AddTunnel(tunnel, host)
		if err != nil {
			s.Logger.Errorf("error when add tunnel: %v", err)
			c.ResponseError(err.Error())
			return
		}

		_ = models.RefreshTunnel(tunnel)
		c.ResponseOk(tunnel)
	}
}

func (s *Service) PutTunnel(c *Context) {
	var form payload.PutTunnelForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		tunnel, err := models.UpdateTunnel(form.Id, form.Mode, form.Source, form.Destination)
		if err != nil {
			s.Logger.Errorf("update tunnel error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		// 先删除隧道再重建
		s.tunnelManager.RemoveTunnel(tunnel.Id)
		err = s.tunnelManager.AddTunnel(tunnel, &tunnel.Host)
		if err != nil {
			s.Logger.Errorf("error when add tunnel, err: %v", err)
			c.ResponseError(err.Error())
			return
		}

		_ = models.RefreshTunnel(tunnel)
		c.ResponseOk(tunnel)
	}
}

func (s *Service) DeleteTunnel(c *Context) {
	var param payload.DeleteTunnelParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeleteTunnelById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete tunnel error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		s.tunnelManager.RemoveTunnel(param.Id)
		c.ResponseOk(nil)
	}
}

// api for table job

func (s *Service) GetJobs(c *Context) {
	var param payload.GetJobsParam
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var jobs []*models.Job
		jobs, err = models.GetAllJob()
		if err != nil {
			s.Logger.Errorf("get jobs error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(jobs)
	}
}

func (s *Service) GetOneJob(c *Context) {
	var param payload.GetJobParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		job, err := models.GetJobById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one job error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(job)
	}
}

func (s *Service) PostJob(c *Context) {
	var form payload.PostJobForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		_, err := parser.Parse(form.Spec)
		if err != nil && form.Type == string(task.JobTypeCron) {
			c.ResponseError(err.Error())
			return
		}
		job, err := models.InsertJob(
			form.Name, form.Type, form.Spec, form.Cmd, form.ExecuteID, form.ExecuteType, form.CmdType)
		if err != nil {
			s.Logger.Errorf("insert job error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		err = s.taskManager.NewJobWithRegister(job, string(task.JobStatusReady))
		if err != nil {
			c.ResponseError(err.Error())
			return
		}

		c.ResponseOk(job)
	}
}

func (s *Service) PutJob(c *Context) {
	var form payload.PutJobForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		_, err := parser.Parse(form.Spec)
		if err != nil && form.Spec != "" && form.Type == string(task.JobTypeCron) {
			c.ResponseError(err.Error())
			return
		}

		job, err := models.UpdateJob(form.Id, form.Name, form.Type, form.Spec, form.Cmd, form.CmdType)
		if err != nil {
			s.Logger.Errorf("update job error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		// 这个错误忽略是为了修改时候只要确认停止即可
		_ = s.taskManager.UnRegister(form.Id, true)

		err = s.taskManager.NewJobWithRegister(job, job.Status)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(job)
	}
}

func (s *Service) DeleteJob(c *Context) {
	var param payload.DeleteJobParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err = s.taskManager.RemoveJob(param.Id)
		if err != nil {
			s.Logger.Errorf("error when remove job, err: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}

// api for table task instance

func (s *Service) GetInstances(c *Context) {
	var (
		total int64
		param payload.GetTaskInstanceParam
	)
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var instances []*models.TaskInstance
		if param.JobId != 0 {
			total, err = models.GetPaginateQuery[*[]*models.TaskInstance](
				&instances, param.PageSize, param.PageNum, map[string]interface{}{
					"job_id": param.JobId,
				}, false)
		} else {
			total, err = models.GetPaginateQuery[*[]*models.TaskInstance](
				&instances, param.PageSize, param.PageNum, nil, false)
		}
		if err != nil {
			s.Logger.Errorf("get instances error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(payload.PageData{
			Data:    instances,
			Total:   total,
			PageNum: param.PageNum,
		})
	}
}

func (s *Service) DeleteInstances(c *Context) {
	var (
		param payload.DeleteTaskInstanceFrom
	)
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var since time.Time

		if param.TimeStamp != 0 {
			since = time.Unix(param.TimeStamp, 0)
		} else {
			since = time.Now().Local().Add(-s.conf.TempDate)
		}

		err = models.ClearInstance(since, param.JobId)
		if err != nil {
			c.ResponseError(err.Error())
		}
		c.ResponseOk(nil)
	}
}

// api for player scheme

func (s *Service) CacheUpload(c *Context) {
	var (
		resp    payload.UploadResponse
		tmpPath = filepath.Join(s.conf.DataPath, config.UploadPath)
	)

	form, err := c.MultipartForm()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	files := form.File["files"]

	if len(files) == 0 {
		c.ResponseError(err.Error())
		return
	}

	for _, file := range files {
		ret := payload.File{
			Name:   file.Filename,
			Size:   file.Size,
			Status: true,
		}
		open, err := file.Open()
		if err != nil {
			ret.Status = false
			continue
		}
		newPath := filepath.Join(tmpPath, uuid.NewString()+file.Filename)
		newFile, err := os.OpenFile(newPath, os.O_RDWR|os.O_CREATE, fs.ModePerm)
		if err != nil {
			ret.Status = false
			continue
		}

		_, err = io.Copy(newFile, open)
		if err != nil {
			ret.Status = false
			continue
		}

		ret.CachePath = newPath

		resp.Files = append(resp.Files, ret)
	}

	c.ResponseOk(resp)
}

func (s *Service) GetPluginSchema(c *Context) {
	c.ResponseOk(s.sshManager.GetAllPluginSchema())
}

func (s *Service) GetPlayBooks(c *Context) {
	records, err := models.GetAllPlayBook()
	if err != nil {
		s.Logger.Errorf("get all playbook error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(records)
}

func (s *Service) GetOnePlayBook(c *Context) {
	var param payload.GetPlayBookParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		record, err := models.GetPlayBookById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one playbook error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(record)
	}
}

func (s *Service) PostPlayBook(c *Context) {
	var form payload.PostPlayBookForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var (
			caches []string
			steps  []*models.Step // 前端定义这个step
		)
		err := json.Unmarshal([]byte(form.Steps), &steps)
		if err != nil {
			c.ResponseError("can not parse steps")
			return
		}
		for _, step := range steps {
			st := s.sshManager.NewStep(step.Type)
			err := json.Unmarshal([]byte(step.Params), st)
			if err != nil {
				s.Logger.Errorf("error when parse plugin param: %s, err: %v", step.Params, err)
				continue
			}

			caches = st.ParseCaches(st)
			mal, _ := json.Marshal(caches)
			step.Caches = string(mal)
		}

		rSteps, _ := json.Marshal(steps)

		record, err := models.InsertPlayBook(form.Name, string(rSteps))
		if err != nil {
			s.Logger.Errorf("insert playbook error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		record.StepsObj = steps

		c.ResponseOk(record)
	}
}

func (s *Service) PutPlayBook(c *Context) {
	var form payload.PutPlayBookForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var (
			caches []string
			steps  []*models.Step // 前端定义这个step
		)
		err := json.Unmarshal([]byte(form.Steps), &steps)
		if err != nil {
			c.ResponseError("can not parse steps")
			return
		}
		for _, step := range steps {
			st := s.sshManager.NewStep(step.Type)
			err := json.Unmarshal([]byte(step.Params), st)
			if err != nil {
				s.Logger.Errorf("error when parse plugin param: %s, err: %v", step.Params, err)
				continue
			}

			caches = st.ParseCaches(st)
			mal, _ := json.Marshal(caches)
			step.Caches = string(mal)
		}

		rSteps, _ := json.Marshal(steps)

		record, err := models.UpdatePlayBook(form.Id, form.Name, string(rSteps))
		if err != nil {
			s.Logger.Errorf("update playbook error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		record.StepsObj = steps

		c.ResponseOk(record)
	}
}

func (s *Service) DeletePlayBook(c *Context) {
	var param payload.DeletePlayBookParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeletePlayBookById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete playbook error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}
