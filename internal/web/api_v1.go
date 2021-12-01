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
	"oms/internal/utils"
	"os"
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

var parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

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
	host, err := models.GetHostByIdWithPreload(id)
	if err != nil {
		s.logger.Errorf("GetOneHost error when GetHostById, err: %v", err)
		data = generateResponsePayload(HttpStatusError, "can not get hosts", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, host)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostHost(c *gin.Context) {
	var tagArray []int
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
	privateKeyId, _ := strconv.Atoi(c.PostForm("private_key_id"))

	if privateKeyId != 0 {
		passFlag = true
	}
	if !passFlag {
		data := generateResponsePayload(HttpStatusError, "must have one of password or private_key_id", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	tags := c.PostForm("tags")
	err := json.Unmarshal([]byte(tags), &tagArray)
	if err != nil {
		s.logger.Errorf("PostHost error when parse tagArray, tags: %s, err: %v", tags, err)
	}

	host, err := models.InsertHost(hostname, user, addr, port, password, groupId, tagArray, privateKeyId)
	if err != nil {
		s.logger.Errorf("PostHost error when InsertHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, host)

	c.JSON(http.StatusOK, data)
}

func (s *Service) PutHost(c *gin.Context) {
	var tagArray []int
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

	privateKeyId, _ := strconv.Atoi(c.PostForm("private_key_id"))

	tags := c.PostForm("tags")
	_ = json.Unmarshal([]byte(tags), &tagArray)

	host, err := models.UpdateHost(id, hostname, user, addr, port, password, groupId, tagArray, privateKeyId)
	if err != nil {
		s.logger.Errorf("PutHost error when UpdateHost, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// 清理缓存的client
	defer s.sshManager.RemoveCache(host)

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
	host, err := models.DeleteHostById(id)
	if err != nil {
		s.logger.Errorf("DeleteHost error when DeleteHostById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete host", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// 清理缓存的client
	defer s.sshManager.RemoveCache(host)

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}

// api for table PrivateKey

func (s *Service) GetPrivateKeys(c *gin.Context) {
	privateKeys, err := models.GetAllPrivateKey()
	if err != nil {
		s.logger.Errorf("GetHosts error when GetAllPrivateKey, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get all privateKey", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, privateKeys)

	c.JSON(http.StatusOK, data)
}

func (s *Service) GetOnePrivateKey(c *gin.Context) {
	var data Response
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("GetOnePrivateKey error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data = generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	privateKey, err := models.GetPrivateKeyById(id)
	if err != nil {
		s.logger.Errorf("GetOnePrivateKey error when GetPrivateKeyById, err: %v", err)
		data = generateResponsePayload(HttpStatusError, "can not get privateKey", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, privateKey)
	c.JSON(http.StatusOK, data)
}

func (s *Service) PostPrivateKey(c *gin.Context) {
	var keyRaw string
	name := c.PostForm("name")
	if name == "" {
		data := generateResponsePayload(HttpStatusError, "name can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	passphrase := c.PostForm("passphrase")

	fh, err := c.FormFile("key_file")
	if err == nil {
		ff, err := fh.Open()
		if err != nil {
			s.logger.Errorf("PostPrivateKey error when Open FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when open key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			s.logger.Errorf("PostPrivateKey error when ReadAll FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when read key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		keyRaw = string(fileBytes)
	}

	if keyRaw == "" {
		data := generateResponsePayload(HttpStatusError, "add an empty key", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	privateKey, err := models.InsertPrivateKey(name, keyRaw, passphrase)
	if err != nil {
		s.logger.Errorf("PostPrivateKey error when InsertPrivateKey, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when create privateKey", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, privateKey)

	c.JSON(http.StatusOK, data)
}

func (s *Service) PutPrivateKey(c *gin.Context) {
	var keyRaw string

	idRaw := c.PostForm("id")
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		s.logger.Errorf("PutHost error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}

	name := c.PostForm("name")
	passphrase := c.PostForm("passphrase")

	fh, err := c.FormFile("key_file")
	if err == nil {
		ff, err := fh.Open()
		if err != nil {
			s.logger.Errorf("PostPrivateKey error when Open FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when open key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			s.logger.Errorf("PostPrivateKey error when ReadAll FormFile, err: %v", err)
			data := generateResponsePayload(HttpStatusError, "error when read key file", nil)
			c.JSON(http.StatusOK, data)
			return
		}
		keyRaw = string(fileBytes)
	}

	privateKey, err := models.UpdatePrivateKey(id, name, keyRaw, passphrase)
	if err != nil {
		s.logger.Errorf("PutPrivateKey error when UpdatePrivateKey, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "error when update privateKey", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, privateKey)

	c.JSON(http.StatusOK, data)
}

func (s *Service) DeletePrivateKey(c *gin.Context) {
	idRaw := c.Param("id")
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		s.logger.Errorf("DeletePrivateKey error when Atoi idRaw, idRaw: %s, err: %v", idRaw, err)
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = models.DeletePrivateKeyById(id)
	if err != nil {
		s.logger.Errorf("DeletePrivateKey error when DeletePrivateKeyById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not delete private_key", nil)
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
	var err error
	var tunnels []*models.Tunnel

	idRaw := c.Query("host_id")
	hostId, _ := strconv.Atoi(idRaw)
	if hostId > 0 {
		tunnels, err = models.GetTunnelsByHostId(hostId)
	} else {
		tunnels, err = models.GetAllTunnel()
	}
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
	if !utils.IsAddr(src) {
		data := generateResponsePayload(HttpStatusError, "src not an address", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dest := c.PostForm("destination")
	if dest == "" {
		data := generateResponsePayload(HttpStatusError, "dest can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	if !utils.IsAddr(dest) {
		data := generateResponsePayload(HttpStatusError, "dest not an address", nil)
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

	_ = models.RefreshTunnel(tunnel)

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
	if src != "" && !utils.IsAddr(src) {
		data := generateResponsePayload(HttpStatusError, "src not an address", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	dest := c.PostForm("destination")
	if dest != "" && !utils.IsAddr(dest) {
		data := generateResponsePayload(HttpStatusError, "dest not an address", nil)
		c.JSON(http.StatusOK, data)
		return
	}
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

	_ = models.RefreshTunnel(tunnel)

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
	var err error
	var jobs []*models.Job

	idRaw := c.Query("host_id")
	hostId, _ := strconv.Atoi(idRaw)
	if hostId > 0 {
		jobs, err = models.GetJobsByHostId(hostId)
	} else {
		jobs, err = models.GetAllJob()
	}

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
	if err != nil && dType == string(task.JobTypeCron) {
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
	dType := c.PostForm("type")
	spec := c.PostForm("spec")
	_, err = parser.Parse(spec)
	if err != nil && spec != "" && dType == string(task.JobTypeCron) {
		msg := fmt.Sprintf("PutJob got a error spec, err: %s", err.Error())
		s.logger.Error(msg)
		data := generateResponsePayload(HttpStatusError, msg, nil)
		c.JSON(http.StatusOK, data)
		return
	}
	cmd := c.PostForm("cmd")
	job, err := models.UpdateJob(id, name, dType, spec, cmd)
	if err != nil {
		s.logger.Errorf("PutJob error when add job, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not create job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// 这个错误忽略是为了修改时候只要确认停止即可
	_ = s.taskManager.UnRegister(id, false)

	err = s.taskManager.NewJobWithRegister(job, job.Status)
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

	_ = models.RefreshJob(job)

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
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
	job, err := models.GetJobById(id)
	if err != nil {
		s.logger.Errorf("StartJob error when GetJobById, err: %v", err)
		data := generateResponsePayload(HttpStatusError, "can not get job", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.taskManager.StopJob(job.Id)
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}

	_ = models.RefreshJob(job)

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
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

	_ = models.RefreshJob(job)

	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, job)
	c.JSON(http.StatusOK, data)
}

func (s *Service) GetLogStream(c *gin.Context) {
	var offset int64
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

	_, _ = fmt.Fprintln(w, fmt.Sprintf("[job]: %s, [cmd]: %s log\n", job.Name(), job.Cmd()))
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
			if line == nil {
				continue
			}
			_, err := fmt.Fprintln(w, line.Text)
			if err != nil {
				return
			}
			w.(http.Flusher).Flush()
		case <-w.CloseNotify():
			s.logger.Debug("log stream got done notify.")
			return
		}
	}
}
