package web

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/fs"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
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
	if cmd == "" {
		data := generateResponsePayload(HttpStatusError, "cmd can not be empty", nil)
		c.JSON(http.StatusOK, data)
		return
	}
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

	if file != nil {
		defer file.Close()

		fh, err := file.Stat()
		if err != nil {
			s.logger.Errorf("error when Stat file, err: %v", err)
			c.String(http.StatusOK, "file stat error")
			return
		}

		mode := fh.Mode() & fs.ModeType
		// this is a read able file
		if mode == 0 {
			extraHeaders := map[string]string{
				"Content-Disposition": fmt.Sprintf("inline; filename=\"%s\"", fh.Name()),
			}

			reader := bufio.NewReader(file)

			if fh.Size() != 0 {
				c.DataFromReader(http.StatusOK, fh.Size(), "text/plain", reader, extraHeaders)
				return
			} else {
				buf, err := ioutil.ReadAll(file)
				if err != nil {
					s.logger.Errorf("read virtual file error: %v", err)
				}
				http.ServeContent(c.Writer, c.Request, fh.Name(), fh.ModTime(), bytes.NewReader(buf))
			}
		} else {
			c.String(http.StatusOK, fmt.Sprintf("read error, file type is [%s]", mode))
		}
	} else {
		c.String(http.StatusOK, "file not existed")
		return
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
		data := generateResponsePayload(HttpStatusError, "ExportData error when export data", nil)
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

func (s *Service) FileUploadV2(c *gin.Context) {
	// todo 使用steam传输
	mediaType, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		c.String(http.StatusOK, "parse content type error")
		return
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		partReader := multipart.NewReader(c.Request.Body, params["boundary"])
		buf := make([]byte, 4096)
		for {
			part, err := partReader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Println(part.FormName())
			if part.FileName() == "" {
				all, _ := ioutil.ReadAll(part)
				fmt.Println(string(all))
			} else {
				fmt.Println(part.FileName())
				for {
					_, err = part.Read(buf)
					if errors.Is(err, io.EOF) {
						break
					}
				}
			}
		}
	}
	c.String(http.StatusOK, "done")
}
