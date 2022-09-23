package web

import (
	_ "embed"
	"fmt"
	staticF "github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"strings"

	"mime"
	"net/http"
	"oms/docs"
	"oms/internal/config"
	"oms/internal/ssh"
	"oms/internal/task"
	"oms/internal/tunnel"
	"oms/internal/web/controllers"
	"oms/web"
	"os/exec"
	"runtime"
)

func CORS(ctx *gin.Context) {
	method := ctx.Request.Method

	// set response header
	ctx.Header("Access-Control-Allow-Origin", ctx.Request.Header.Get("Origin"))
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers",
		"Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With, X-Files")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

	if method == "OPTIONS" || method == "HEAD" {
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}

	ctx.Next()
}

// exportHeaders export header Content-Disposition for axios
func exportHeaders(ctx *gin.Context) {
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Next()
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

type HandlerFunc func(c *controllers.Context)

func Handle(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &controllers.Context{
			Context: c,
		}
		h(ctx)
	}
}

func InitRouter(s *controllers.Service) *controllers.Service {
	r := gin.New()

	r.Use(gin.Recovery()).Use(CORS).Use(exportHeaders)

	// swagger docs
	docs.SwaggerInfo.BasePath = "/api/v1"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	err := mime.AddExtensionType(".js", "application/javascript")
	if err != nil {
		s.Logger.Errorf("error when add extension type, err: %v", err)
	}

	// metrics
	r.GET("/metrics", prometheusHandler())

	static := &web.ServeFileSystem{
		E:    web.EmbeddedFiles,
		Path: "omsUI/dist/assets",
	}

	r.Use(staticF.Serve("/assets", static))

	r.Use(gin.Logger())
	// common api
	r.GET("/", s.GetIndexPage)

	//if not route (route from frontend) redirect to index
	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.RequestURI, "/api") {
			s.GetIndexPage(c)
		}
	})

	// websocket
	r.GET("/ws/index", s.GetWebsocketIndex)
	r.GET("/ws/ssh/:id", s.GetWebsocketSSH)
	r.GET("/ws/vnc/:id", s.GetWebsocketVNC)

	// restapi
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/host", Handle(s.GetHosts))
		apiV1.GET("/host/:id", Handle(s.GetOneHost))
		apiV1.POST("/host", Handle(s.PostHost))
		apiV1.PUT("/host", Handle(s.PutHost))
		apiV1.DELETE("/host/:id", Handle(s.DeleteHost))

		apiV1.GET("/private_key", Handle(s.GetPrivateKeys))
		apiV1.GET("/private_key/:id", Handle(s.GetOnePrivateKey))
		apiV1.POST("/private_key", Handle(s.PostPrivateKey))
		apiV1.PUT("/private_key", Handle(s.PutPrivateKey))
		apiV1.DELETE("/private_key/:id", Handle(s.DeletePrivateKey))

		apiV1.GET("/group", Handle(s.GetGroups))
		apiV1.GET("/group/:id", Handle(s.GetOneGroup))
		apiV1.POST("/group", Handle(s.PostGroup))
		apiV1.PUT("/group", Handle(s.PutGroup))
		apiV1.DELETE("/group/:id", Handle(s.DeleteGroup))

		apiV1.GET("/tag", Handle(s.GetTags))
		apiV1.GET("/tag/:id", Handle(s.GetOneTag))
		apiV1.POST("/tag", Handle(s.PostTag))
		apiV1.PUT("/tag", Handle(s.PutTag))
		apiV1.DELETE("/tag/:id", Handle(s.DeleteTag))

		apiV1.GET("/tunnel", Handle(s.GetTunnels))
		apiV1.GET("/tunnel/:id", Handle(s.GetOneTunnel))
		apiV1.POST("/tunnel", Handle(s.PostTunnel))
		apiV1.PUT("/tunnel", Handle(s.PutTunnel))
		apiV1.DELETE("/tunnel/:id", Handle(s.DeleteTunnel))

		apiV1.GET("/job", Handle(s.GetJobs))
		apiV1.GET("/job/:id", Handle(s.GetOneJob))
		apiV1.POST("/job", Handle(s.PostJob))
		apiV1.PUT("/job", Handle(s.PutJob))
		apiV1.DELETE("/job/:id", Handle(s.DeleteJob))
		//apiV1.GET("/job/tail", Handle(s.GetLogStream))
		apiV1.POST("/job/exec", Handle(s.ExecJob))
		apiV1.POST("/job/start", Handle(s.StartJob))
		apiV1.POST("/job/stop", Handle(s.StopJob))
		apiV1.GET("/task/instance", Handle(s.GetInstances))
		apiV1.DELETE("/task/instance", Handle(s.DeleteInstances))
		apiV1.GET("/task/instance/log/download", Handle(s.DownloadInstanceLog))
		apiV1.GET("/task/instance/log/get", Handle(s.GetInstanceLog))

		// command
		apiV1.GET("/command/history", Handle(s.GetCommandHistory))
		apiV1.DELETE("/command/history/:id", Handle(s.DeleteCommandHistory))

		apiV1.GET("/quick_command", Handle(s.GetQuicklyCommand))
		apiV1.GET("/quick_command/:id", Handle(s.GetOneQuicklyCommand))
		apiV1.POST("/quick_command", Handle(s.PostQuicklyCommand))
		apiV1.PUT("/quick_command", Handle(s.PutQuicklyCommand))
		apiV1.DELETE("/quick_command/:id", Handle(s.DeleteQuicklyCommand))

		// tools
		apiV1.GET("/tools/cmd", Handle(s.RunCmd))
		apiV1.GET("/tools/preview", Handle(s.FilePreview))
		apiV1.GET("/tools/browse", Handle(s.GetPathInfo))
		apiV1.POST("/tools/mkdir", Handle(s.MakeDirRemote))
		apiV1.GET("/tools/download", Handle(s.DownLoadFile))
		apiV1.POST("/tools/delete", Handle(s.DeleteFile))
		apiV1.POST("/tools/upload_file", Handle(s.FileUploadUnBlock))
		apiV1.GET("/tools/export", Handle(s.DataExport))
		apiV1.POST("/tools/import", Handle(s.DataImport))

		// steam version
		apiV1.POST("/tools/upload", Handle(s.FileUploadV2))

		// player scheme
		apiV1.GET("/schema", Handle(s.GetPluginSchema))
		apiV1.POST("/cache/upload", Handle(s.CacheUpload))
		apiV1.GET("/player", Handle(s.GetPlayBooks))
		apiV1.GET("/player/:id", Handle(s.GetOnePlayBook))
		apiV1.POST("/player", Handle(s.PostPlayBook))
		apiV1.PUT("/player", Handle(s.PutPlayBook))
		apiV1.DELETE("/player/:id", Handle(s.DeletePlayBook))
		apiV1.POST("/plugin/upload", Handle(s.PluginUpload))

		apiV1.POST("/player/import", Handle(s.PlayerImport))
		apiV1.GET("/player/export", Handle(s.PlayerExport))

		// version
		apiV1.GET("/version", Handle(s.GetVersion))
	}
	s.Engine = r

	return s
}

func Serve(conf config.App, sshManager *ssh.Manager, taskManager *task.Manager, tunnelManager *tunnel.Manager) {
	gin.SetMode(gin.ReleaseMode)
	s := InitRouter(controllers.NewService(conf, sshManager, taskManager, tunnelManager))

	s.Logger.Infof("Listening and serving HTTP on %s", s.Addr)
	go func() {
		if err := s.Engine.Run(s.Addr); err != nil {
			panic(err)
		}
	}()

	// 配置管理是否在运行时启动浏览器
	if !conf.RunStart {
		return
	}
	// todo mac
	if runtime.GOOS == "windows" {
		urlPath := fmt.Sprintf("http://127.0.0.1:%d", conf.Port)
		cmd := exec.Command("cmd", "/c", "start", urlPath)
		err := cmd.Start()
		if err != nil {
			s.Logger.Errorf("start server in browser error: %v", err)
			return
		}
	}
}
