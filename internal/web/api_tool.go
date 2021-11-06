package web

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io/fs"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func (s *Service) RunCmd(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	pType := c.Query("type")
	cmd := c.Query("cmd")
	sudoRaw := c.Query("sudo")
	// default false
	sudo, _ := strconv.ParseBool(sudoRaw)

	hosts := s.ParseHostList(pType, id)
	if len(hosts) == 0 {
		data := generateResponsePayload(HttpStatusError, "parse host array empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	// do cmd
	results := s.RunCmdExec(hosts, cmd, sudo)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)

}

func (s *Service) FileUpload(c *gin.Context) {
	var remoteFile string
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
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

	results := s.UploadFile(hosts, files, remoteFile)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)

}

func (s *Service) FileUploadUnBlock(c *gin.Context) {
	var remoteFile string
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
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
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	c.JSON(http.StatusOK, data)

}

func (s *Service) GetPathInfo(c *gin.Context) {
	path := c.Query("path")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, err.Error(), nil)
		c.JSON(http.StatusOK, data)
		return
	}
	results := s.GetPathInfoExec(id, path)
	data := generateResponsePayload(HttpStatusOk, HttpResponseSuccess, results)
	c.JSON(http.StatusOK, data)
}

func (s *Service) DownLoadFile(c *gin.Context) {
	path := c.Query("path")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	file := s.DownloadFile(id, path)
	defer file.Close()

	if file != nil {
		fh, err := file.Stat()
		mode := fh.Mode() & fs.ModeType
		if mode == 0 {
			// proc
			buf, _ := ioutil.ReadAll(file)
			http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), bytes.NewReader(buf))
		}

		if err != nil {
			s.logger.Errorf("DownLoadFile error when Stat file, err: %v", err)
			http.ServeContent(c.Writer, c.Request, "download", time.Now(), file)
		} else {
			http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), file)
		}
	} else {
		data := generateResponsePayload(HttpStatusError, "download file error", nil)
		c.JSON(http.StatusOK, data)
	}
}

func (s *Service) DeleteFile(c *gin.Context) {
	var data Response
	path := c.PostForm("path")
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "can not parse param id", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	err = s.DeleteFileOrDir(id, path)
	if err != nil {
		data = generateResponsePayload(HttpStatusError, "remove file error", nil)
	} else {
		data = generateResponsePayload(HttpStatusOk, HttpResponseSuccess, nil)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Service) ExportData(c *gin.Context) {
	marshal, err := s.ExportDbData()
	if err != nil {
		data := generateResponsePayload(HttpStatusError, "error export data", nil)
		c.JSON(http.StatusOK, data)
		return
	}
	file := bytes.NewReader(marshal)
	c.Writer.Header().Set("content-type", "application/json")
	http.ServeContent(c.Writer, c.Request, "export", time.Now(), file)
}

func (s *Service) ImportData(c *gin.Context) {
	fh, err := c.FormFile("dataFile")
	if err == nil {
		ff, _ := fh.Open()
		fileBytes, err := ioutil.ReadAll(ff)
		if err != nil {
			s.logger.Error(err)
			data := generateResponsePayload(HttpStatusError, err.Error(), nil)
			c.JSON(http.StatusOK, data)
			return
		}
		err = s.ImportDbData(fileBytes)
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
