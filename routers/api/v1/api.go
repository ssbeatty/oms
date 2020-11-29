package v1

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"oms/logger"
	"oms/models"
	"strconv"
)

const (
	HttpStatusOk    = "200"
	HttpStatusError = "400"
)

type ResponseGet struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ResponsePost struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// api for table host
func GetHosts(c *gin.Context) {
	hosts := models.GetAllHost()
	data := &ResponseGet{HttpStatusOk, "success",
		hosts}

	c.JSON(http.StatusOK, data)
}

func GetOneHost(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	hosts := models.GetHostById(id)
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	data := &ResponseGet{code, msg,
		hosts}
	c.JSON(http.StatusOK, data)
}

func PostHost(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	var tagJson []string
	var keyText string
	hostname := c.PostForm("hostname")
	user := c.PostForm("user")
	addr := c.PostForm("addr")
	port, err := strconv.Atoi(c.DefaultPostForm("port", "0"))
	if port == 0 {
		port = 22
	}
	password := c.PostForm("password")
	groupId, err := strconv.Atoi(c.PostForm("group"))

	fh, err := c.FormFile("keyFile")
	tags := c.PostForm("tags")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			logger.Logger.Println(err)
		}
		keyText = string(fileBytes)
	}

	err = json.Unmarshal([]byte(tags), &tagJson)
	if err != nil {
		logger.Logger.Println(err)
	}

	models.InsertHost(hostname, user, addr, port, password, groupId, tagJson, keyText)
	data := &ResponsePost{code, msg}

	c.JSON(http.StatusOK, data)
}

func PutHost(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	var tagJson []string
	var keyText string
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	hostname := c.PostForm("hostname")
	user := c.PostForm("user")
	addr := c.PostForm("addr")
	port, _ := strconv.Atoi(c.PostForm("port"))
	password := c.PostForm("password")
	groupId, _ := strconv.Atoi(c.PostForm("group"))
	fh, err := c.FormFile("keyFile")

	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			logger.Logger.Println(err)
		}
		keyText = string(fileBytes)
	}
	tags := c.PostForm("tags")

	err = json.Unmarshal([]byte(tags), &tagJson)
	if err != nil {
		msg = "can't get param tags"
		code = HttpStatusError
		logger.Logger.Println(err)
	}

	models.UpdateHost(id, hostname, user, addr, port, password, groupId, tagJson, keyText)
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

func DeleteHost(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	result := models.DeleteHostById(id)
	if !result {
		msg = "Can't delete object"
		code = HttpStatusError
	}
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

// api for table group
func GetGroups(c *gin.Context) {
	groups := models.GetAllGroup()
	data := &ResponseGet{HttpStatusOk, "success",
		groups}
	c.JSON(http.StatusOK, data)
}

func GetOneGroup(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	group := models.GetGroupById(id)
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	data := &ResponseGet{code, msg,
		group}
	c.JSON(http.StatusOK, data)
}

func PostGroup(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	name := c.PostForm("name")
	params := c.PostForm("params")
	mode, err := strconv.Atoi(c.PostForm("mode"))
	if err != nil {
		msg = "Error params mode"
		code = HttpStatusError
	}
	models.InsertGroup(name, params, mode)
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

func PutGroup(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	name := c.PostForm("name")
	params := c.PostForm("params")
	mode, err := strconv.Atoi(c.PostForm("mode"))
	if err != nil {
		msg = "Error params mode"
		code = HttpStatusError
	}

	models.UpdateGroup(id, name, params, mode)
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

func DeleteGroup(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	result := models.DeleteGroupById(id)
	if !result {
		msg = "Can't delete object"
		code = HttpStatusError
	}
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

// api for table tag
func GetTags(c *gin.Context) {
	tags := models.GetAllTag()
	data := &ResponseGet{HttpStatusOk, "success",
		tags}
	c.JSON(http.StatusOK, data)
}

func GetOneTag(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	tag := models.GetTagById(id)
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	data := &ResponseGet{code, msg,
		tag}
	c.JSON(http.StatusOK, data)
}

func PostTag(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	name := c.PostForm("name")

	models.InsertTag(name)
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

func PutTag(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		logger.Logger.Println(err)
		msg = "can't get param id"
		code = HttpStatusError
	}
	name := c.PostForm("name")

	models.UpdateTag(id, name)
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}

func DeleteTag(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		msg = "Error params id"
		code = HttpStatusError
	}
	result := models.DeleteTagById(id)
	if !result {
		msg = "Can't delete object"
		code = HttpStatusError
	}
	data := &ResponsePost{code, msg}
	c.JSON(http.StatusOK, data)
}
