package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/metrics"
	"github.com/ssbeatty/oms/internal/ssh"
	"github.com/ssbeatty/oms/internal/task"
	"github.com/ssbeatty/oms/internal/tunnel"
	"github.com/ssbeatty/oms/internal/web/payload"
	"github.com/ssbeatty/oms/pkg/logger"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

type Service struct {
	Engine        *gin.Engine
	Addr          string
	Logger        *logger.Logger
	conf          config.App
	taskManager   *task.Manager
	tunnelManager *tunnel.Manager
	sshManager    *ssh.Manager
	metrics       *metrics.Manager
}

func NewService(conf config.App, sshManager *ssh.Manager, taskManager *task.Manager, tunnelManager *tunnel.Manager) *Service {
	service := &Service{
		Addr:          fmt.Sprintf("%s:%d", conf.Addr, conf.Port),
		sshManager:    sshManager,
		taskManager:   taskManager,
		tunnelManager: tunnelManager,
		metrics:       metrics.NewManager(sshManager, taskManager, tunnelManager).Init(),
		Logger:        logger.NewLogger("web"),
		conf:          conf,
	}

	return service
}

type Context struct {
	*gin.Context
}

func (c *Context) ResponseError(msg string) {
	d := payload.GenerateErrorResponse(HttpStatusError, msg)
	c.JSON(http.StatusOK, d)
}

func (c *Context) ResponseOk(data interface{}) {
	d := payload.GenerateDataResponse(HttpStatusOk, HttpResponseSuccess, data)
	c.JSON(http.StatusOK, d)
}

func (c *Context) readFormFile(header *multipart.FileHeader) ([]byte, error) {
	ff, err := header.Open()
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadAll(ff)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}
