package v1

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"oms/models"
	"strconv"
	"time"
)

func RunCmd(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		log.Println(err)
	}
	pType := c.Query("type")
	cmd := c.Query("cmd")
	hosts := models.ParseHostList(pType, id)
	// do cmd
	results := models.RunCmd(hosts, cmd)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)

}

func FileUpload(c *gin.Context) {
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
		log.Println(err)
	}
	pType := c.PostForm("type")
	hosts := models.ParseHostList(pType, id)

	results := models.UploadFile(hosts, files, remoteFile)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)

}
func GetPathInfo(c *gin.Context) {
	path := c.Query("path")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	results := models.GetPathInfo(id, path)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)
}

func DownLoadFile(c *gin.Context) {
	path := c.Query("path")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		log.Println(err)
	}
	file := models.DownloadFile(id, path)
	if file != nil {
		fh, err := file.Stat()
		if err != nil {
			log.Println(err)
			http.ServeContent(c.Writer, c.Request, "download", time.Now(), file)
		} else {
			http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), file)
		}
	} else {
		data := generateResponsePayload(HttpStatusError, "download file error", nil)
		c.JSON(http.StatusOK, data)
	}
}

func DeleteFile(c *gin.Context) {
	var data Response
	path := c.PostForm("path")
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		log.Println(err)
	}
	err = models.DeleteFileOrDir(id, path)
	if err != nil {
		data = generateResponsePayload(HttpStatusError, "remove file error", nil)
	} else {
		data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	}
	c.JSON(http.StatusOK, data)
}

func ExportData(c *gin.Context) {
	marshal, err := models.ExportDbData()
	if err != nil {
		log.Println(err)
	}
	file := bytes.NewReader(marshal)
	c.Writer.Header().Set("content-type", "application/json")
	http.ServeContent(c.Writer, c.Request, "export", time.Now(), file)
}

func ImportData(c *gin.Context) {
	fh, err := c.FormFile("dataFile")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			log.Println(err)
			data := generateResponsePayload(HttpStatusError, err.Error(), nil)
			c.JSON(http.StatusOK, data)
			return
		}
		err = models.ImportDbData(fileBytes)
		if err != nil {
			data := generateResponsePayload(HttpStatusError, err.Error(), nil)
			c.JSON(http.StatusOK, data)
			return
		}
	} else {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)
}
