package page

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"oms/logger"
	"oms/models"
	"strconv"
	"time"
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

func RunCmd(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	pType := c.Query("type")
	cmd := c.Query("cmd")
	hosts := models.ParseHostList(pType, id)
	// do cmd
	results := models.RunCmd(hosts, cmd)
	data := &ResponseGet{Code: code, Msg: msg, Data: results}
	c.JSON(http.StatusOK, data)

}

func FileUpload(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	var remoteFile string
	id, err := strconv.Atoi(c.PostForm("id"))
	form, _ := c.MultipartForm()
	files := form.File["files"]
	remote := c.PostForm("remote")
	if remote[len(remote)-1] == '/' {
		remoteFile = remote
	} else {
		remoteFile = remote + "/"
	}
	if err != nil {
		logger.Logger.Println(err)
	}
	pType := c.PostForm("type")
	hosts := models.ParseHostList(pType, id)

	results := models.UploadFile(hosts, files, remoteFile)
	data := &ResponseGet{Code: code, Msg: msg, Data: results}
	c.JSON(http.StatusOK, data)

}
func GetPathInfo(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	path := c.Query("path")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		logger.Logger.Println(err)
		code = HttpStatusError
		msg = err.Error()
	}
	results := models.GetPathInfo(id, path)
	data := &ResponseGet{Code: code, Msg: msg, Data: results}
	c.JSON(http.StatusOK, data)
}

func DownLoadFile(c *gin.Context) {
	path := c.Query("path")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	file := models.DownloadFile(id, path)
	if file != nil {
		fh, err := file.Stat()
		if err != nil {
			logger.Logger.Println(err)
			http.ServeContent(c.Writer, c.Request, "download", time.Now(), file)
		} else {
			http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), file)
		}
	} else {
		data := &ResponsePost{Code: HttpStatusError, Msg: "download file error"}
		c.JSON(http.StatusOK, data)
	}
}

func DeleteFile(c *gin.Context) {
	var data *ResponsePost
	path := c.PostForm("path")
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		logger.Logger.Println(err)
	}
	err = models.DeleteFileOrDir(id, path)
	if err != nil {
		data = &ResponsePost{Code: HttpStatusError, Msg: "remove file error"}
	} else {
		data = &ResponsePost{Code: HttpStatusOk, Msg: "success"}
	}
	c.JSON(http.StatusOK, data)
}

func ExportData(c *gin.Context) {
	marshal, err := models.ExportDbData()
	if err != nil {
		logger.Logger.Println(err)
	}
	file := bytes.NewReader(marshal)
	c.Writer.Header().Set("content-type", "application/json")
	http.ServeContent(c.Writer, c.Request, "export", time.Now(), file)
}

func ImportData(c *gin.Context) {
	var msg = "success"
	var code = HttpStatusOk
	fh, err := c.FormFile("dataFile")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			logger.Logger.Println(err)
			msg = err.Error()
			code = HttpStatusError
		}
		err = models.ImportDbData(fileBytes)
		if err != nil {
			logger.Logger.Println(err)
			msg = err.Error()
			code = HttpStatusError
		}
	} else {
		msg = err.Error()
		code = HttpStatusError
	}
	data := &ResponsePost{Code: code, Msg: msg}
	c.JSON(http.StatusOK, data)
}
