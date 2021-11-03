package v1

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"oms/models"
	"oms/pkg/schedule"
	Tunnel "oms/pkg/tunnel"
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

func generateResponsePayload(code string, msg string, data interface{}) Response {
	return Response{code, msg, data}
}

// api for table host

func GetHosts(c *gin.Context) {
	hosts, err := models.GetAllHost()
	if err != nil {
		log.Errorf("GetHosts error when GetAllHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get all host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, hosts)

	c.JSON(http.StatusOK, data)
}

func GetOneHost(c *gin.Context) {
	var data Response
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("GetOneHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data = generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	hosts, err := models.GetHostById(id)
	if err != nil {
		log.Errorf("GetOneHost error when GetHostById, err: %v", err)
		data = generateResponsePayload(HttpStatusError, "can not get hosts", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, hosts)
	c.JSON(http.StatusOK, data)
}

func PostHost(c *gin.Context) {
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
			log.Errorf("PostHost error when Open FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when open key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			log.Errorf("PostHost error when ReadAll FormFile, err: %v", err)
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
		log.Errorf("PostHost error when parse tagArray, tags: %s, err: %v", tags, err)
	}

	host, err := models.InsertHost(hostname, user, addr, port, password, groupId, tagArray, keyRaw)
	if err != nil {
		log.Errorf("PostHost error when InsertHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when create host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, host)

	c.JSON(http.StatusOK, data)
}

func PutHost(c *gin.Context) {
	var tagArray []string
	var keyRaw string
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		log.Errorf("PutHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
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
			log.Errorf("PutHost error when Open FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when open key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			log.Errorf("PutHost error when ReadAll FormFile, err: %v", err)
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
		log.Errorf("PutHost error when UpdateHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, host)
	c.JSON(http.StatusOK, data)
}

func DeleteHost(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("DeleteHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteHostById(id)
	if err != nil {
		log.Errorf("DeleteHost error when DeleteHostById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table group

func GetGroups(c *gin.Context) {
	groups, err := models.GetAllGroup()
	if err != nil {
		log.Errorf("GetGroups error when GetAllGroup, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when get groups", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, groups)
	c.JSON(http.StatusOK, data)
}

func GetOneGroup(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("GetOneGroup error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	group, err := models.GetGroupById(id)
	if err != nil {
		log.Errorf("GetGroups error when GetGroupById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when get group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, group)
	c.JSON(http.StatusOK, data)
}

func PostGroup(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	params := c.PostForm("params")
	mode, err := strconv.Atoi(c.PostForm("mode"))
	if err != nil {
		log.Errorf("PostGroup error when Atoi mode, mode: %d, err: %v", mode, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param mode", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	group, err := models.InsertGroup(name, params, mode)
	if err != nil {
		log.Errorf("PostGroup error when InsertGroup, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when create group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, group)
	c.JSON(http.StatusOK, data)
}

func PutGroup(c *gin.Context) {
	var mode int

	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("PutGroup error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
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
		log.Errorf("PutGroup error when UpdateGroup, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, group)
	c.JSON(http.StatusOK, data)
}

func DeleteGroup(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("DeleteGroup error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteGroupById(id)
	if err != nil {
		log.Errorf("DeleteGroup error when DeleteGroupById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not parse delete group", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table tag

func GetTags(c *gin.Context) {
	tags, err := models.GetAllTag()
	if err != nil {
		log.Errorf("GetTags error when GetAllTag, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tags", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tags)
	c.JSON(http.StatusOK, data)
}

func GetOneTag(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("GetOneTag error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tag, err := models.GetTagById(id)
	if err != nil {
		log.Errorf("GetOneTag error when GetTagById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tag)
	c.JSON(http.StatusOK, data)
}

func PostTag(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tag, err := models.InsertTag(name)
	if err != nil {
		log.Errorf("PostTag error when InsertTag, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tag)
	c.JSON(http.StatusOK, data)
}

func PutTag(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("PutTag error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
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
		log.Errorf("PutTag error when UpdateTag, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not update tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tag)
	c.JSON(http.StatusOK, data)
}

func DeleteTag(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("PutTag error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteTagById(id)
	if err != nil {
		log.Errorf("DeleteTag error when DeleteTagById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete tag", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table tunnel

func GetTunnels(c *gin.Context) {
	tunnels, err := models.GetAllTunnel()
	if err != nil {
		log.Errorf("GetTunnels error when GetAllTunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tunnels", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnels)
	c.JSON(http.StatusOK, data)
}

func GetOneTunnel(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("GetOneTunnel error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tunnel, err := models.GetTunnelById(id)
	if err != nil {
		log.Errorf("GetOneTunnel error when GetTunnelById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnel)
	c.JSON(http.StatusOK, data)
}

func PostTunnel(c *gin.Context) {
	mode := c.PostForm("mode")
	src := c.PostForm("src")
	if src == "" {
		data := generateResponsePayload(HttpStatusError, "src can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dest := c.PostForm("dest")
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
		log.Errorf("PostTunnel error when Atoi idStr, idRaw: %s ,err: %v", hostIdRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	tunnel, err := Tunnel.DefaultManager.AddTunnel(hostId, mode, src, dest)
	if err != nil {
		log.Errorf("error when add tunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnel)
	c.JSON(http.StatusOK, data)
}

func PutTunnel(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("PutTunnel error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	mode := c.PostForm("mode")
	src := c.PostForm("src")
	dest := c.PostForm("dest")
	tunnel, err := models.UpdateTunnel(id, mode, src, dest)
	if err != nil {
		log.Errorf("PutTunnel error when UpdateTunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not update tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// 先删除隧道再重建
	Tunnel.DefaultManager.RemoveTunnel(tunnel.Id)
	tunnel, err = Tunnel.DefaultManager.AddTunnel(tunnel.HostId, mode, src, dest)
	if err != nil {
		log.Errorf("error when add tunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, tunnel)
	c.JSON(http.StatusOK, data)
}

func DeleteTunnel(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeleteTunnelById(id)
	if err != nil {
		log.Errorf("error when DeleteTunnelById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete tunnel", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	Tunnel.DefaultManager.RemoveTunnel(id)

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table job

func GetJobs(c *gin.Context) {
	jobs, err := models.GetAllJob()
	if err != nil {
		log.Errorf("error when GetAllJob, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get jobs", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, jobs)
	c.JSON(http.StatusOK, data)
}

func GetOneJob(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		log.Errorf("error when GetJobById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func PostJob(c *gin.Context) {
	name := c.PostForm("name")
	spec := c.PostForm("spec")
	dType := c.PostForm("type")
	if dType == "" {
		data := generateResponsePayload(HttpStatusError, "type can not be empty", nil)
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
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", hostIdRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	host, err := models.GetHostById(hostId)
	if err != nil {
		log.Errorf("error when get host, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get host", nil)
		c.JSON(http.StatusOK, data)
	}
	job, err := models.InsertJob(name, dType, spec, cmd, host)
	if err != nil {
		log.Errorf("error when add tunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create job", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	schedule.NewJobWithRegister(job, string(schedule.JobStatusReady))

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func PutJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	name := c.PostForm("name")
	spec := c.PostForm("spec")
	dType := c.PostForm("type")
	cmd := c.PostForm("cmd")
	job, err := models.UpdateJob(id, name, dType, spec, cmd)
	if err != nil {
		log.Errorf("error when add tunnel, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = schedule.RemoveJob(id)
	if err != nil {
		log.Errorf("error when RemoveJob, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	schedule.NewJobWithRegister(job, string(schedule.JobStatusReady))

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func DeleteJob(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = schedule.RemoveJob(id)
	if err != nil {
		log.Errorf("error when RemoveJob, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

func StartJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	job, err := models.GetJobById(id)
	if err != nil {
		log.Errorf("error when GetJobById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	schedule.StartJob(job)
}

func StopJob(c *gin.Context) {
	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		log.Errorf("error when Atoi idStr, idRaw: %s ,err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	schedule.StopJob(id)
}
