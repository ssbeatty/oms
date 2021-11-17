package web

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hpcloud/tail"
	"github.com/robfig/cron/v3"
	"io"
	"io/ioutil"
	"net/http"
	"oms/internal/models"
	"oms/internal/task"
	"strconv"
)

const (
	HttpStatusOk        = "200"
	HttpStatusError     = "400"
	HttpResponseSuccess = "success"
)

type Response struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func generateResponsePayload(code string, msg string, data interface{}) Response {
	return Response{code, msg, data}
}

// api for table host

func (s *Service) GetHosts(c *gin.Context) {
	hosts, err := models.GetAllHost()
	if err != nil {
		s.logger.Errorf("GetHosts error when GetAllHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get all host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, hosts)

	c.JSON(http.StatusOK, data)
}

func (s *Service) GetOneHost(c *gin.Context) {
	var data Response
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetOneHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data = generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hosts, err := models.GetHostByIdWithPreload(id)
	if err != nil {
		s.logger.Errorf("GetOneHost error when GetHostById, err: %v", err)
		data = generateResponsePayload(HttpStatusError, "can not get hosts", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, hosts)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostHost(c *gin.Context) {
	var keyRaw string
	var tagArray []string
	var passFlag bool

	hostname := c.PostForm("hostname")
	user := c.PostForm("user")
	addr := c.PostForm("addr")
	if hostname == "" || user == "" || addr == "" {
		data := generateResponsePayload(HttpStatusError, "hostname, user and addr can not empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	port, _ := strconv.Atoi(c.DefaultPostForm("port", "0"))
	if port == 0 {
		port = 22
	}
	password := c.PostForm("password")
	if password != "" {
		passFlag = true
	}
	// if not group param, it is zero
	groupId, _ := strconv.Atoi(c.PostForm("group"))

	fh, err := c.FormFile("keyFile")
	if err == nil {
		ff, err := fh.Open()
		if err != nil {
			s.logger.Errorf("PostHost error when Open FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when open key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			s.logger.Errorf("PostHost error when ReadAll FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when read key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		keyRaw = string(fileBytes)
		passFlag = true
	}

	if !passFlag {
		data := generateResponsePayload(HttpStatusError, "must have one of password or keyfile", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	tags := c.PostForm("tags")
	err = json.Unmarshal([]byte(tags), &tagArray)
	if err != nil {
		s.logger.Errorf("PostHost error when parse tagArray, tags: %s, err: %v", tags, err)
	}

	host, err := models.InsertHost(hostname, user, addr, port, password, groupId, tagArray, keyRaw)
	if err != nil {
		s.logger.Errorf("PostHost error when InsertHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when create host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, host)

	c.JSON(http.StatusOK, data)
}

func (s *Service) PutHost(c *gin.Context) {
	var tagArray []string
	var keyRaw string
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		s.logger.Errorf("PutHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hostname := c.PostForm("hostname")
	user := c.PostForm("user")
	addr := c.PostForm("addr")
	port, _ := strconv.Atoi(c.PostForm("port"))
	password := c.PostForm("password")
	groupId, _ := strconv.Atoi(c.PostForm("group"))
	fh, err := c.FormFile("keyFile")

	if err == nil {
		ff, err := fh.Open()
		if err != nil {
			s.logger.Errorf("PutHost error when Open FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when open key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			s.logger.Errorf("PutHost error when ReadAll FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when read key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		keyRaw = string(fileBytes)
	}

	tags := c.PostForm("tags")
	_ = json.Unmarshal([]byte(tags), &tagArray)

	host, err := models.UpdateHost(id, hostname, user, addr, port, password, groupId, tagArray, keyRaw)
	if err != nil {
		s.logger.Errorf("PutHost error when UpdateHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, host)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DeleteHost(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("DeleteHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteHostById(id)
	if err != nil {
		s.logger.Errorf("DeleteHost error when DeleteHostById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table group

func (s *Service) GetGroups(c *gin.Context) {
	groups, err := models.GetAllGroup()
	if err != nil {
		s.logger.Errorf("GetGroups error when GetAllGroup, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when get groups", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, groups)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetOneGroup(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetOneGroup error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	group, err := models.GetGroupById(id)
	if err != nil {
		s.logger.Errorf("GetOneGroup error when GetGroupById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when get group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, group)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostGroup(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	params := c.PostForm("params")
	mode, err := strconv.Atoi(c.PostForm("mode"))
	if err != nil {
		s.logger.Errorf("PostGroup error when Atoi mode, mode: %d, err: %v", mode, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param mode", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	group, err := models.InsertGroup(name, params, mode)
	if err != nil {
		s.logger.Errorf("PostGroup error when InsertGroup, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when create group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, group)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PutGroup(c *gin.Context) {
	var mode int

	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("PutGroup error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	name := c.PostForm("name")
	params := c.PostForm("params")
	modeRaw, err := strconv.Atoi(c.PostForm("mode"))
	if err != nil {
		mode = -1
	} else {
		mode = modeRaw
	}

	group, err := models.UpdateGroup(id, name, params, mode)
	if err != nil {
		s.logger.Errorf("PutGroup error when UpdateGroup, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, group)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DeleteGroup(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("DeleteGroup error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteGroupById(id)
	if err != nil {
		s.logger.Errorf("DeleteGroup error when DeleteGroupById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not parse delete group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table tag

func (s *Service) GetTags(c *gin.Context) {
	tags, err := models.GetAllTag()
	if err != nil {
		s.logger.Errorf("GetTags error when GetAllTag, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tags", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tags)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetOneTag(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetOneTag error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tag, err := models.GetTagById(id)
	if err != nil {
		s.logger.Errorf("GetOneTag error when GetTagById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tag)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostTag(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tag, err := models.InsertTag(name)
	if err != nil {
		s.logger.Errorf("PostTag error when InsertTag, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tag)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PutTag(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("PutTag error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tag, err := models.UpdateTag(id, name)
	if err != nil {
		s.logger.Errorf("PutTag error when UpdateTag, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not update tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tag)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DeleteTag(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("DeleteTag error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteTagById(id)
	if err != nil {
		s.logger.Errorf("DeleteTag error when DeleteTagById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table tunnel

func (s *Service) GetTunnels(c *gin.Context) {
	tunnels, err := models.GetAllTunnel()
	if err != nil {
		s.logger.Errorf("GetTunnels error when GetAllTunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tunnels", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnels)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetOneTunnel(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetOneTunnel error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tunnel, err := models.GetTunnelById(id)
	if err != nil {
		s.logger.Errorf("GetOneTunnel error when GetTunnelById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnel)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostTunnel(c *gin.Context) {
	mode := c.PostForm("mode")
	src := c.PostForm("source")
	if src == "" {
		data := generateResponsePayload(HttpStatusError, "src can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dest := c.PostForm("destination")
	if dest == "" {
		data := generateResponsePayload(HttpStatusError, "dest can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hostIdRaw := c.PostForm("host_id")
	if hostIdRaw == "" {
		data := generateResponsePayload(HttpStatusError, "host_id can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hostId, err := strconv.Atoi(hostIdRaw)
	if err != nil {
		s.logger.Errorf("PostTunnel error when Atoi idStr, idRaw: %s ,err: %v", hostIdRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	host, err := models.GetHostById(hostId)
	if err != nil {
		s.logger.Errorf("PostTunnel error when GetHostById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tunnel, err := models.InsertTunnel(mode, src, dest, host)
	if err != nil {
		s.logger.Errorf("PostTunnel error when InsertTunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tunnel model", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	err = s.tunnelManager.AddTunnel(tunnel, host)
	if err != nil {
		s.logger.Errorf("PostTunnel error when add tunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnel)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PutTunnel(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("PutTunnel error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	mode := c.PostForm("mode")
	src := c.PostForm("source")
	dest := c.PostForm("destination")
	tunnel, err := models.UpdateTunnel(id, mode, src, dest)
	if err != nil {
		s.logger.Errorf("PutTunnel error when UpdateTunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not update tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	host, err := models.GetHostById(tunnel.HostId)
	if err != nil {
		s.logger.Errorf("PostTunnel error when GetHostById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// 先删除隧道再重建
	s.tunnelManager.RemoveTunnel(tunnel.Id)
	err = s.tunnelManager.AddTunnel(tunnel, host)
	if err != nil {
		s.logger.Errorf("PutTunnel error when add tunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnel)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DeleteTunnel(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("DeleteTunnel error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteTunnelById(id)
	if err != nil {
		s.logger.Errorf("DeleteTunnel error when DeleteTunnelById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	s.tunnelManager.RemoveTunnel(id)

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table job

func (s *Service) GetJobs(c *gin.Context) {
	jobs, err := models.GetAllJob()
	if err != nil {
		s.logger.Errorf("GetJobs error when GetAllJob, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get jobs", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, jobs)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetOneJob(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetOneJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		s.logger.Errorf("GetOneJob error when GetJobById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostJob(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dType := c.PostForm("type")
	if dType == "" {
		data := generateResponsePayload(HttpStatusError, "type can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	spec := c.PostForm("spec")
	if dType == string(task.JobTypeCron) && spec == "" {
		data := generateResponsePayload(HttpStatusError, "spec can not be empty if type is cron", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	_, err := parser.Parse(spec)
	if err != nil {
		msg := fmt.Sprintf("PostJob got a error spec, err: %s", err.Error())
		s.logger.Error(msg)
		data := generateResponsePayload(HttpStatusError, msg, nil)
		c.JSON(http.StatusOK, data)
		return
	}
	cmd := c.PostForm("cmd")
	if cmd == "" {
		data := generateResponsePayload(HttpStatusError, "cmd can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hostIdRaw := c.PostForm("host_id")
	if hostIdRaw == "" {
		data := generateResponsePayload(HttpStatusError, "host_id can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hostId, err := strconv.Atoi(hostIdRaw)
	if err != nil {
		s.logger.Errorf("PostJob error when Atoi idStr, idRaw: %s ,err: %v", hostIdRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param host id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	host, err := models.GetHostById(hostId)
	if err != nil {
		s.logger.Errorf("PostJob error when GetHostById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get host", nil)
		c.JSON(http.StatusOK, data)
	}
	job, err := models.InsertJob(name, dType, spec, cmd, host)
	if err != nil {
		s.logger.Errorf("PostJob error when InsertJob, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create job", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	err = s.taskManager.NewJobWithRegister(job, string(task.JobStatusReady))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PutJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("PutJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	name := c.PostForm("name")
	spec := c.PostForm("spec")
	_, err = parser.Parse(spec)
	if err != nil && spec != "" {
		msg := fmt.Sprintf("PutJob got a error spec, err: %s", err.Error())
		s.logger.Error(msg)
		data := generateResponsePayload(HttpStatusError, msg, nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dType := c.PostForm("type")
	cmd := c.PostForm("cmd")
	job, err := models.UpdateJob(id, name, dType, spec, cmd)
	if err != nil {
		s.logger.Errorf("PutJob error when add job, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.UnRegister(id)
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.NewJobWithRegister(job, string(task.JobStatusReady))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DeleteJob(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("DeleteJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.RemoveJob(id)
	if err != nil {
		s.logger.Errorf("DeleteJob error when RemoveJob, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func (s *Service) StartJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("StartJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		s.logger.Errorf("StartJob error when GetJobById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.StartJob(job)
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func (s *Service) StopJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("StopJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.StopJob(id)
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func (s *Service) RestartJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("RestartJob error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		s.logger.Errorf("RestartJob error when GetJobById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	_ = s.taskManager.StopJob(id)

	err = s.taskManager.StartJob(job)
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetLogStream(c *gin.Context) {
	idRaw := c.Query("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetLogStream error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	job, ok := s.taskManager.GetJob(id)
	if !ok {
		data := generateResponsePayload(HttpStatusError, "can not found job", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	w := c.Writer
	header := w.Header()
	//http chunked
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Content-Type", "text/plain")

	w.Write([]byte(fmt.Sprintf("[job]: %s, [cmd]: %s log\n", job.Name(), job.Cmd())))
	w.(http.Flusher).Flush()

	t, err := tail.TailFile(job.Log(), tail.Config{
		Follow:   true,
		Poll:     true,
		Location: &tail.SeekInfo{Offset: -2000, Whence: io.SeekEnd},
	})
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "tail file error", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	defer func() {
		err := t.Stop()
		if err != nil {
			s.logger.Errorf("error when stop tail, err: %v", err)
			return
		}
		s.logger.Debug("GetLogStream log file stream exit.")
	}()

	for {
		select {
		case line := <-t.Lines:
			_, err := w.Write([]byte(line.Text))
			if err != nil {
				return
			}
			w.(http.Flusher).Flush()
		case <-w.CloseNotify():
			return
		}
	}
}
