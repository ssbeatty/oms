package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"html/template"
	"mime"
	"net/http"
	"oms/internal/config"
	"oms/internal/metrics"
	"oms/internal/ssh"
	"oms/internal/task"
	"oms/internal/tunnel"
	"oms/pkg/logger"
	"os/exec"
	"runtime"
)

type Service struct {
	engine *gin.Engine
	addr   string
	port   int
	logger *logger.Logger

	taskManager   *task.Manager
	tunnelManager *tunnel.Manager
	sshManager    *ssh.Manager
	metrics       *metrics.Manager
}

func NewService(conf config.App, sshManager *ssh.Manager, taskManager *task.Manager, tunnelManager *tunnel.Manager) *Service {
	service := &Service{
		addr:          fmt.Sprintf("%s:%d", conf.Addr, conf.Port),
		port:          conf.Port,
		sshManager:    sshManager,
		taskManager:   taskManager,
		tunnelManager: tunnelManager,
		metrics:       metrics.NewManager(sshManager, taskManager, tunnelManager).Init(),
		logger:        logger.NewLogger("web"),
	}

	return service
}

func CORS(ctx *gin.Context) {
	method := ctx.Request.Method

	// set response header
	ctx.Header("Access-Control-Allow-Origin", ctx.Request.Header.Get("Origin"))
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With, X-Files")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

	// 默认过滤这两个请求,使用204(No Content)这个特殊的http status code
	if method == "OPTIONS" || method == "HEAD" {
		ctx.AbortWithStatus(204)
		return
	}

	ctx.Next()
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func (s *Service) InitRouter() *Service {
	r := gin.Default()
	r.Use(CORS)

	// static files
	box := packr.NewBox("../../web/omsUI/dist/assets")
	r.StaticFS("/assets", box)

	err := mime.AddExtensionType(".js", "application/javascript")
	if err != nil {
		s.logger.Errorf("error when add extension type, err: %v", err)
	}

	// load template
	t, err := loadTemplate()
	if err != nil {
		panic("error when load template.")
	}
	r.SetHTMLTemplate(t)

	// metrics
	r.GET("/metrics", prometheusHandler())

	// common api
	r.GET("/", GetIndexPage)
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	// websocket
	r.GET("/ws/index", s.GetWebsocketIndex)
	r.GET("/ws/ssh/:id", s.GetWebsocketSsh)

	// restapi
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/host", s.GetHosts)
		apiV1.GET("/host/:id", s.GetOneHost)
		apiV1.POST("/host", s.PostHost)
		apiV1.PUT("/host", s.PutHost)
		apiV1.DELETE("/host/:id", s.DeleteHost)

		apiV1.GET("/group", s.GetGroups)
		apiV1.GET("/group/:id", s.GetOneGroup)
		apiV1.POST("/group", s.PostGroup)
		apiV1.PUT("/group", s.PutGroup)
		apiV1.DELETE("/group/:id", s.DeleteGroup)

		apiV1.GET("/tag", s.GetTags)
		apiV1.GET("/tag/:id", s.GetOneTag)
		apiV1.POST("/tag", s.PostTag)
		apiV1.PUT("/tag", s.PutTag)
		apiV1.DELETE("/tag/:id", s.DeleteTag)

		apiV1.GET("/tunnel", s.GetTunnels)
		apiV1.GET("/tunnel/:id", s.GetOneTunnel)
		apiV1.POST("/tunnel", s.PostTunnel)
		apiV1.PUT("/tunnel", s.PutTunnel)
		apiV1.DELETE("/tunnel/:id", s.DeleteTunnel)

		apiV1.GET("/job", s.GetJobs)
		apiV1.GET("/job/:id", s.GetOneJob)
		apiV1.POST("/job", s.PostJob)
		apiV1.PUT("/job", s.PutJob)
		apiV1.DELETE("/job/:id", s.DeleteJob)
		apiV1.GET("/job/tail", s.GetLogStream)
		apiV1.POST("/job/start", s.StartJob)
		apiV1.POST("/job/stop", s.StopJob)
		apiV1.POST("/job/restart", s.RestartJob)

		// tools
		apiV1.GET("/tools/cmd", s.RunCmd)
		apiV1.GET("/tools/browse", s.GetPathInfo)
		apiV1.GET("/tools/download", s.DownLoadFile)
		apiV1.POST("/tools/delete", s.DeleteFile)
		apiV1.GET("/tools/export", s.ExportData)
		apiV1.POST("/tools/import", s.ImportData)
		apiV1.POST("/tools/upload_file", s.FileUploadUnBlock)

		// steam version
		apiV1.POST("/tools/upload", s.FileUploadV2)
	}
	s.engine = r
	return s
}

func (s *Service) Run() {
	s.logger.Infof("Listening and serving HTTP on %s", s.addr)
	go func() {
		if err := s.engine.Run(s.addr); err != nil {
			panic(err)
		}
	}()

	// todo mac
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", fmt.Sprintf("http://127.0.0.1:%d", s.port))
		err := cmd.Start()
		if err != nil {
			s.logger.Errorf("start server in browser error: %v", err)
			return
		}
	}
}

func (s *Service) SetRelease() {
	gin.SetMode(gin.ReleaseMode)
}

func loadTemplate() (*template.Template, error) {
	const indexFile = "index.html"
	box := packr.NewBox("../../web/omsUI/dist")
	t := template.New("")
	data, err := box.Find(indexFile)
	if err != nil {
		return nil, err
	}
	t, err = t.New(indexFile).Parse(string(data))
	if err != nil {
		return nil, err
	}
	return t, nil
}
