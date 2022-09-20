package controllers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"io"
	"io/fs"
	"io/ioutil"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/task"
	"oms/internal/web/payload"
	"oms/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HttpStatusOk        = "200"
	HttpStatusError     = "400"
	HttpResponseSuccess = "success"
)

var parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// @BasePath /api/v1

// GetHosts
// @Summary 获取所有主机
// @Description 获取所有主机
// @Param page_num query int false  "页码数"
// @Param page_size query int false  "分页尺寸" default(20)
// @Tags host
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.Host}
// @Failure 400 {object} payload.Response
// @Router /host [get]
func (s *Service) GetHosts(c *Context) {
	var (
		hosts []*models.Host
		param payload.GetAllHostParam
	)
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		total, err := models.GetPaginateQuery[*[]*models.Host](
			&hosts, param.PageSize, param.PageNum, nil, true)
		if err != nil {
			s.Logger.Errorf("get all host error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(payload.PageData{
			Data:    hosts,
			Total:   total,
			PageNum: param.PageNum,
		})
	}
}

// GetOneHost
// @Summary 获取单个主机
// @Description 获取单个主机
// @Param id path int true  "主机 ID"
// @Tags host
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Host}
// @Failure 400 {object} payload.Response
// @Router /host/{id} [get]
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

// PostHost
// @Summary 创建主机
// @Description 创建主机
// @Param hostname formData string true "主机名称"
// @Param user formData string true "用户名"
// @Param addr formData string true "地址"
// @Param port formData integer true "SSH端口"
// @Param password formData string false "SSH密码"
// @Param group formData integer false "组ID"
// @Param private_key_id formData integer false "密钥ID"
// @Param tags formData string false "标签ID列表序列化字符串"
// @Param vnc_port formData integer false "VNC端口"
// @Tags host
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Host}
// @Failure 400 {object} payload.Response
// @Router /host [post]
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

// PutHost
// @Summary 更新主机
// @Description 更新主机
// @Param id formData integer true "主机 ID"
// @Param hostname formData string false "主机名称"
// @Param user formData string false "用户名"
// @Param addr formData string false "地址"
// @Param port formData integer false "SSH端口"
// @Param password formData string false "SSH密码"
// @Param group formData integer false "组ID"
// @Param private_key_id formData integer false "密钥ID"
// @Param tags formData string false "标签ID列表序列化字符串"
// @Param vnc_port formData integer false "VNC端口"
// @Tags host
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Host}
// @Failure 400 {object} payload.Response
// @Router /host [put]
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

// DeleteHost
// @Summary 删除主机
// @Description 删除主机
// @Param id path int true  "主机 ID"
// @Tags host
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /host/{id} [delete]
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

// GetPrivateKeys
// @Summary 获取所有密钥
// @Description 获取所有密钥
// @Tags private_key
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.PrivateKey}
// @Failure 400 {object} payload.Response
// @Router /private_key [get]
func (s *Service) GetPrivateKeys(c *Context) {
	privateKeys, err := models.GetAllPrivateKey()
	if err != nil {
		s.Logger.Errorf("get all privateKey error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(privateKeys)
}

// GetOnePrivateKey
// @Summary 获取单个密钥
// @Description 获取单个密钥
// @Param id path int true  "密钥 ID"
// @Tags private_key
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.PrivateKey}
// @Failure 400 {object} payload.Response
// @Router /private_key/{id} [get]
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

// PostPrivateKey
// @Summary 创建密钥
// @Description 创建密钥
// @Param name formData string true "密钥名称"
// @Param passphrase formData string false "密钥密码"
// @Param key_file formData file true "密钥文件"
// @Tags private_key
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.PrivateKey}
// @Failure 400 {object} payload.Response
// @Router /private_key [post]
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

// PutPrivateKey
// @Summary 更新密钥
// @Description 更新密钥
// @Param id formData integer true "密钥 ID"
// @Param name formData string false "密钥名称"
// @Param passphrase formData string false "密钥密码"
// @Param key_file formData file false "密钥文件"
// @Tags private_key
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.PrivateKey}
// @Failure 400 {object} payload.Response
// @Router /private_key [put]
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

// DeletePrivateKey
// @Summary 删除密钥
// @Description 删除密钥
// @Param id path int true  "密钥 ID"
// @Tags private_key
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /private_key/{id} [delete]
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

// GetGroups
// @Summary 获取所有组
// @Description 获取所有组
// @Tags group
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.Group}
// @Failure 400 {object} payload.Response
// @Router /group [get]
func (s *Service) GetGroups(c *Context) {
	groups, err := models.GetAllGroup()
	if err != nil {
		s.Logger.Errorf("get all group error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(groups)
}

// GetOneGroup
// @Summary 获取单个组
// @Description 获取单个组
// @Param id path int true  "组 ID"
// @Tags group
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Group}
// @Failure 400 {object} payload.Response
// @Router /group/{id} [get]
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

// PostGroup
// @Summary 创建组
// @Description 创建组
// @Param name formData string true "组名称"
// @Param params formData string false "组参数"
// @Param mode formData int true "组类型" example(0:主机模式,1:匹配模式)
// @Tags group
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Group}
// @Failure 400 {object} payload.Response
// @Router /group [post]
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

// PutGroup
// @Summary 更新组
// @Description 更新组
// @Param id formData integer true "组 ID"
// @Param name formData string false "组名称"
// @Param params formData string false "组参数"
// @Param mode formData int false "组类型" example(0:主机模式,1:匹配模式)
// @Tags group
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Group}
// @Failure 400 {object} payload.Response
// @Router /group [put]
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

// DeleteGroup
// @Summary 删除组
// @Description 删除组
// @Param id path int true  "组 ID"
// @Tags group
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /group/{id} [delete]
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

// GetTags
// @Summary 获取所有标签
// @Description 获取所有标签
// @Tags tag
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.Tag}
// @Failure 400 {object} payload.Response
// @Router /tag [get]
func (s *Service) GetTags(c *Context) {
	tags, err := models.GetAllTag()
	if err != nil {
		s.Logger.Errorf("get all tag error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(tags)
}

// GetOneTag
// @Summary 获取单个标签
// @Description 获取单个标签
// @Param id path int true  "标签 ID"
// @Tags tag
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Tag}
// @Failure 400 {object} payload.Response
// @Router /tag/{id} [get]
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

// PostTag
// @Summary 创建标签
// @Description 创建标签
// @Param name formData string true "标签名称"
// @Tags tag
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Tag}
// @Failure 400 {object} payload.Response
// @Router /tag [post]
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

// PutTag
// @Summary 更新标签
// @Description 更新标签
// @Param id formData integer true "标签 ID"
// @Param name formData string false "标签名称"
// @Tags tag
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Tag}
// @Failure 400 {object} payload.Response
// @Router /tag [put]
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

// DeleteTag
// @Summary 删除标签
// @Description 删除标签
// @Param id path int true  "标签 ID"
// @Tags tag
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tag/{id} [delete]
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

// GetTunnels
// @Summary 获取所有隧道
// @Description 获取所有隧道
// @Tags tunnel
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.Tunnel}
// @Failure 400 {object} payload.Response
// @Router /tunnel [get]
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

// GetOneTunnel
// @Summary 获取单个隧道
// @Description 获取单个隧道
// @Param id path int true  "隧道 ID"
// @Tags tunnel
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Tunnel}
// @Failure 400 {object} payload.Response
// @Router /tunnel/{id} [get]
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

// PostTunnel
// @Summary 创建隧道
// @Description 创建隧道
// @Param mode formData string true "隧道模式" example(local,remote)
// @Param source formData string true "源地址"
// @Param destination formData int true "目的地址"
// @Param host_id formData int true "主机 ID"
// @Tags tunnel
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Tunnel}
// @Failure 400 {object} payload.Response
// @Router /tunnel [post]
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

// PutTunnel
// @Summary 更新隧道
// @Description 更新隧道
// @Param id formData integer true "隧道 ID"
// @Param mode formData string false "隧道模式" example(local,remote)
// @Param source formData string false "源地址"
// @Param destination formData int false "目的地址"
// @Param host_id formData int false "主机 ID"
// @Tags tunnel
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Tunnel}
// @Failure 400 {object} payload.Response
// @Router /tunnel [put]
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

// DeleteTunnel
// @Summary 删除隧道
// @Description 删除隧道
// @Param id path int true  "隧道 ID"
// @Tags group
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /tunnel/{id} [delete]
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

// GetJobs
// @Summary 获取所有任务
// @Description 获取所有任务
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.Job}
// @Failure 400 {object} payload.Response
// @Router /job [get]
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

// GetOneJob
// @Summary 获取单个任务
// @Description 获取单个任务
// @Param id path int true  "任务 ID"
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Job}
// @Failure 400 {object} payload.Response
// @Router /job/{id} [get]
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

// PostJob
// @Summary 创建任务
// @Description 创建任务
// @Param name formData string true "任务名称"
// @Param type formData string true "任务类型" example(cron,local)
// @Param spec formData string false "Cron表达式"
// @Param cmd formData string false "任务命令"
// @Param cmd_id formData integer false "剧本ID"
// @Param cmd_type formData string true "任务命令类型" example(cmd,player)
// @Param execute_id formData integer true "执行者 ID"
// @Param execute_type formData string true "执行者类型" example(host,group,tag)
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Job}
// @Failure 400 {object} payload.Response
// @Router /job [post]
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
		if form.CmdType == ssh.CMDTypePlayer && form.CmdId == 0 {
			c.ResponseError("cmd_id can not null")
			return
		}
		job, err := models.InsertJob(
			form.Name, form.Type, form.Spec, form.Cmd, form.ExecuteID, form.CmdId, form.ExecuteType, form.CmdType)
		if err != nil {
			s.Logger.Errorf("insert job error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		_, err = s.taskManager.NewJobWithRegister(job, string(task.JobStatusReady))
		if err != nil {
			c.ResponseError(err.Error())
			return
		}

		c.ResponseOk(job)
	}
}

// PutJob
// @Summary 更新任务
// @Description 更新任务
// @Param id formData integer true "任务 ID"
// @Param name formData string false "任务名称"
// @Param type formData string false "任务类型" example(cron,local)
// @Param spec formData string false "Cron表达式"
// @Param cmd formData string false "任务命令"
// @Param cmd_id formData integer false "剧本ID"
// @Param cmd_type formData string false "任务命令类型" example(cmd,player)
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.Job}
// @Failure 400 {object} payload.Response
// @Router /job [put]
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

		job, err := models.UpdateJob(form.Id, form.Name, form.Type, form.Spec, form.Cmd, form.CmdType, form.CmdId, form.ExecuteID, form.ExecuteType)
		if err != nil {
			s.Logger.Errorf("update job error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		// 这个错误忽略是为了修改时候只要确认停止即可
		_ = s.taskManager.UnRegister(form.Id, false)

		_, err = s.taskManager.NewJobWithRegister(job, job.Status)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(job)
	}
}

// DeleteJob
// @Summary 删除任务
// @Description 删除任务
// @Param id path int true  "任务 ID"
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /job/{id} [delete]
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

// GetInstances
// @Summary 获取所有任务执行结果
// @Description 获取所有任务执行结果
// @Param job_id query int false  "任务 ID"
// @Param page_num query int false  "页码数"
// @Param page_size query int false  "分页尺寸" default(20)
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.TaskInstance}
// @Failure 400 {object} payload.Response
// @Router /task/instance [get]
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

// DeleteInstances
// @Summary 删除任务执行结果
// @Description 删除任务执行结果
// @Param job_id formData integer false "任务 ID"
// @Param time_stamp formData integer false "截止时间戳(之前的数据都将删除)"
// @Tags job
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /task/instance [delete]
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

// CacheUpload
// @Summary 上传临时文件
// @Description 上传临时文件
// @Param files formData file true "multi文件"
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /cache/upload [post]
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
		c.ResponseError(errors.New("empty files").Error())
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

// GetPluginSchema
// @Summary 获取所有插件Schema
// @Description 获取所有插件Schema
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]ssh.Schema}
// @Failure 400 {object} payload.Response
// @Router /schema [get]
func (s *Service) GetPluginSchema(c *Context) {
	c.ResponseOk(s.sshManager.GetAllPluginSchema())
}

// GetPlayBooks
// @Summary 获取所有剧本
// @Description 获取所有剧本
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.PlayBook}
// @Failure 400 {object} payload.Response
// @Router /player [get]
func (s *Service) GetPlayBooks(c *Context) {
	records, err := models.GetAllPlayBook()
	if err != nil {
		s.Logger.Errorf("get all playbook error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(records)
}

// GetOnePlayBook
// @Summary 获取单个剧本
// @Description 获取单个剧本
// @Param id path int true  "剧本 ID"
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.PlayBook}
// @Failure 400 {object} payload.Response
// @Router /player/{id} [get]
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

// PostPlayBook
// @Summary 创建剧本
// @Description 创建剧本
// @Param name formData string true "剧本名称"
// @Param steps formData string true "剧本步骤序列化字符串" example([{"seq":0,"type":"cmd","name":"执行ls","caches":"null","params":"{\"cmd\":\"ls\"}"}])
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.PlayBook}
// @Failure 400 {object} payload.Response
// @Router /player [post]
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

// PutPlayBook
// @Summary 更新剧本
// @Description 更新剧本
// @Param id formData integer true "剧本 ID"
// @Param name formData string false "剧本名称"
// @Param steps formData string false "剧本步骤序列化字符串" example([{"seq":0,"type":"cmd","name":"执行ls","caches":"null","params":"{\"cmd\":\"ls\"}"}])
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.PlayBook}
// @Failure 400 {object} payload.Response
// @Router /player [put]
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

// DeletePlayBook
// @Summary 删除剧本
// @Description 删除剧本
// @Param id path int true  "剧本 ID"
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /player/{id} [delete]
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

// PluginUpload
// @Summary 上传插件
// @Description 上传插件
// @Param files formData file true "插件文件(zip or tar.gz)"
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /plugin/upload [post]
func (s *Service) PluginUpload(c *Context) {
	var (
		pluginPath = filepath.Join(s.conf.DataPath, config.PluginPath)
	)

	form, err := c.MultipartForm()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	files := form.File["files"]

	if len(files) == 0 {
		c.ResponseError("empty files")
		return
	}

	for _, file := range files {
		fh, err := file.Open()
		if err != nil {
			s.Logger.Errorf("error when open file %s, err: %v", file.Filename, err)
			continue
		}

		switch utils.GetFileExt(file.Filename) {
		case "zip":
			var (
				zipRootDir string
			)

			reader, err := zip.NewReader(fh, file.Size)
			if err != nil {
				s.Logger.Errorf("error when create zip reader %s, err: %v", file.Filename, err)
				continue
			}
			if len(reader.File) == 1 && reader.File[0].FileInfo().IsDir() {
				zipRootDir = reader.File[0].Name
			} else {
				zipRootDir = strings.TrimRight(file.Filename, "zip")
			}

			zipRootPath := filepath.Join(pluginPath, zipRootDir)

			err = os.MkdirAll(zipRootPath, fs.ModePerm)
			if err != nil {
				s.Logger.Errorf("error when create tmp dir %s, err: %v", file.Filename, err)
				continue
			}

			for _, f := range reader.File {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				filename := filepath.Join(zipRootPath, f.Name)
				err = os.MkdirAll(filepath.Dir(filename), fs.ModePerm)
				if err != nil {
					continue
				}
				w, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, fs.ModePerm)
				if err != nil {
					continue
				}
				_, err = io.Copy(w, rc)
				if err != nil {
					continue
				}
				w.Close()
				rc.Close()
			}
		case "tar.gz":
			var (
				tarFiles   []*tar.Header
				zipRootDir string
			)

			gr, err := gzip.NewReader(fh)
			if err != nil {
				s.Logger.Errorf("error when create gzip reader %s, err: %v", file.Filename, err)
				continue
			}

			tr := tar.NewReader(gr)
			for {
				hdr, err := tr.Next()
				if err != nil || err == io.EOF {
					break
				}
				if hdr == nil {
					continue
				}

				tarFiles = append(tarFiles, hdr)
			}
			if len(tarFiles) == 1 && tarFiles[0].FileInfo().IsDir() {
				zipRootDir = tarFiles[0].Name
			} else {
				zipRootDir = strings.TrimRight(file.Filename, "tar.gz")
			}

			zipRootPath := filepath.Join(pluginPath, zipRootDir)

			for _, f := range tarFiles {
				dstFileDir := filepath.Join(zipRootPath, f.Name)

				switch f.Typeflag {
				case tar.TypeDir:
					if ok, _ := utils.PathExists(dstFileDir); !ok {
						if err := os.MkdirAll(dstFileDir, fs.ModePerm); err != nil {
							continue
						}
					}
				case tar.TypeReg:
					if ok, _ := utils.PathExists(filepath.Dir(dstFileDir)); !ok {
						if err := os.MkdirAll(dstFileDir, fs.ModePerm); err != nil {
							continue
						}
					}
					_file, err := os.OpenFile(dstFileDir, os.O_CREATE|os.O_RDWR, fs.ModePerm)
					if err != nil {
						continue
					}
					_, err = io.Copy(_file, tr)
					if err != nil {
						continue
					}
					_file.Close()
				}
			}

			gr.Close()
		default:
			c.ResponseError("unsupported file format")
			return
		}

		fh.Close()

		s.sshManager.ReloadAllFilePlugins(pluginPath)
	}

	c.ResponseOk(nil)
}

// PlayerImport
// @Summary 导入剧本
// @Description 导入剧本
// @Param files formData file true "剧本导出文件(zip)"
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /player/import [post]
func (s *Service) PlayerImport(c *Context) {
	var (
		uploadPath = filepath.Join(s.conf.DataPath, config.UploadPath)
	)

	form, err := c.MultipartForm()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	files := form.File["files"]

	if len(files) == 0 {
		c.ResponseError("empty files")
		return
	}

	for _, file := range files {
		if utils.GetFileExt(file.Filename) != "zip" {
			c.ResponseError("unsupported file")
			return
		}
		fh, err := file.Open()
		if err != nil {
			s.Logger.Errorf("error when open file %s, err: %v", file.Filename, err)
			continue
		}

		reader, err := zip.NewReader(fh, file.Size)
		if err != nil {
			s.Logger.Errorf("error when create zip reader %s, err: %v", file.Filename, err)
			continue
		}
		for _, f := range reader.File {
			fn, err := f.Open()
			if err != nil {
				continue
			}

			switch f.Name {
			case "metadata.json":
				playerMap := make(map[string]string)
				data, err := ioutil.ReadAll(fn)
				if err != nil {
					continue
				}

				err = json.Unmarshal(data, &playerMap)
				if err != nil {
					c.ResponseError(err.Error())
					return
				}

				for k, val := range playerMap {
					if models.ExistedPlayBook(k, val) {
						continue
					}
					_, err := models.InsertPlayBook(k, val)
					if err != nil {
						continue
					}
				}
			default:
				baseName := filepath.Base(f.Name)

				dstFile, err := os.OpenFile(filepath.Join(uploadPath, baseName), os.O_CREATE|os.O_TRUNC|os.O_RDWR, fs.ModePerm)
				if err != nil {
					continue
				}

				_, _ = io.Copy(dstFile, fn)

				dstFile.Close()
			}

			fn.Close()
		}

		fh.Close()
	}

	c.ResponseOk(nil)
}

// PlayerExport
// @Summary 导出剧本
// @Description 导出剧本
// @Tags player
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200
// @Failure 400 {object} payload.Response
// @Router /player/export [get]
func (s *Service) PlayerExport(c *Context) {
	const (
		subPath = "upload"
	)

	var (
		metaPlayer = make(map[string]string)
		pluginPath = filepath.Join(s.conf.DataPath, config.DefaultTmpPath)
	)

	tmpName := fmt.Sprintf("%d-players.zip", time.Now().Unix())
	tmpPath := filepath.Join(pluginPath, tmpName)
	tmpFile, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE, fs.ModePerm)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	defer os.Remove(tmpPath)

	players, err := models.GetAllPlayBook()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	writer := zip.NewWriter(tmpFile)

	for _, player := range players {
		metaPlayer[player.Name] = player.Steps
		for _, step := range player.StepsObj {
			if len(step.GetCaches()) == 0 {
				continue
			}
			for _, cache := range step.GetCaches() {
				zipFile, err := writer.Create(filepath.ToSlash(filepath.Join(subPath, filepath.Base(cache))))
				if err != nil {
					continue
				}
				osFile, err := os.OpenFile(cache, os.O_RDONLY, fs.ModePerm)
				if err != nil {
					continue
				}

				_, err = io.Copy(zipFile, osFile)
				if err != nil {
					return
				}

				osFile.Close()
			}
		}
	}
	metaFile, err := writer.Create("metadata.json")
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	data, err := json.Marshal(metaPlayer)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	_, err = metaFile.Write(data)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	writer.Close()
	tmpFile.Close()

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s", tmpName))
	c.File(tmpPath)
}

// GetCommandHistory
// @Summary 模糊查询历史命令
// @Description 模糊查询历史命令
// @Param keyword query string true "关键词"
// @Param limit query integer false "limit"
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]string}
// @Failure 400 {object} payload.Response
// @Router /command/history [get]
func (s *Service) GetCommandHistory(c *Context) {
	var (
		param payload.SearchCmdHistoryParams
	)
	err := c.ShouldBind(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		var (
			ret     []string
			records []*models.CommandHistory
		)
		records, err = models.SearchCommandHistory(param.KeyWord, param.Limit)
		if err != nil {
			s.Logger.Errorf("search cmd history error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		for _, v := range records {
			ret = append(ret, v.Cmd)
		}
		c.ResponseOk(ret)
	}
}

// DeleteCommandHistory
// @Summary 删除命令历史
// @Description 删除命令历史
// @Param id path int true  "命令历史 ID"
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /command/history/{id} [delete]
func (s *Service) DeleteCommandHistory(c *Context) {
	var param payload.DeleteCmdHistoryParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeleteCommandHistoryById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete cmd history error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}

// GetQuicklyCommand
// @Summary 获取所有快捷命令
// @Description 获取所有快捷命令
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=[]models.QuicklyCommand}
// @Failure 400 {object} payload.Response
// @Router /quick_command [get]
func (s *Service) GetQuicklyCommand(c *Context) {
	records, err := models.GetAllQuicklyCommand()
	if err != nil {
		s.Logger.Errorf("get all quickly command error: %v", err)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(records)
}

// GetOneQuicklyCommand
// @Summary 获取单个快捷命令
// @Description 获取单个快捷命令
// @Param id path int true  "快捷命令 ID"
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.QuicklyCommand}
// @Failure 400 {object} payload.Response
// @Router /quick_command/{id} [get]
func (s *Service) GetOneQuicklyCommand(c *Context) {
	var param payload.GetQuicklyCommandParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		record, err := models.GetQuicklyCommandById(param.Id)
		if err != nil {
			s.Logger.Errorf("get one quickly command error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(record)
	}
}

// PostQuicklyCommand
// @Summary 创建快捷命令
// @Description 创建快捷命令
// @Param name formData string true "快捷命令名称"
// @Param cmd formData string true "快捷命令文本"
// @Param append_cr formData bool false "是否追加CR"
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.QuicklyCommand}
// @Failure 400 {object} payload.Response
// @Router /quick_command [post]
func (s *Service) PostQuicklyCommand(c *Context) {
	var form payload.PostQuicklyCommandForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		record, err := models.InsertQuicklyCommand(form.Name, form.Cmd, form.AppendCR)
		if err != nil {
			s.Logger.Errorf("insert quickly command error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(record)
	}
}

// PutQuicklyCommand
// @Summary 更新快捷命令
// @Description 更新快捷命令
// @Param id formData integer true "快捷命令 ID"
// @Param name formData string false "快捷命令名称"
// @Param cmd formData string false "快捷命令文本"
// @Param append_cr formData bool false "是否追加CR"
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response{data=models.QuicklyCommand}
// @Failure 400 {object} payload.Response
// @Router /quick_command [put]
func (s *Service) PutQuicklyCommand(c *Context) {
	var form payload.PutQuicklyCommandForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		record, err := models.UpdateQuicklyCommand(form.Id, form.Name, form.Cmd, form.AppendCR)
		if err != nil {
			s.Logger.Errorf("update quickly command error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(record)
	}
}

// DeleteQuicklyCommand
// @Summary 删除快捷命令
// @Description 删除快捷命令
// @Param id path int true  "快捷命令 ID"
// @Tags command
// @Accept x-www-form-urlencoded
// @Produce json
// @Success 200 {object} payload.Response
// @Failure 400 {object} payload.Response
// @Router /quick_command/{id} [delete]
func (s *Service) DeleteQuicklyCommand(c *Context) {
	var param payload.DeleteQuicklyCommandParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		c.ResponseError(err.Error())
	} else {
		err := models.DeleteQuicklyCommandById(param.Id)
		if err != nil {
			s.Logger.Errorf("delete quickly command error: %v", err)
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil)
	}
}
