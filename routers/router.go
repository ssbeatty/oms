package routers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"oms/conf"
	v1 "oms/routers/api/v1"
	"oms/routers/page"
)

func CORS(ctx *gin.Context) {
	method := ctx.Request.Method

	// set response header
	ctx.Header("Access-Control-Allow-Origin", ctx.Request.Header.Get("Origin"))
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

	// 默认过滤这两个请求,使用204(No Content)这个特殊的http status code
	if method == "OPTIONS" || method == "HEAD" {
		ctx.AbortWithStatus(204)
		return
	}

	ctx.Next()
}

func InitGinServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	//r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// static files
	box := packr.NewBox("../static")
	r.StaticFS("/static", box)
	t, _ := loadTemplate()
	r.SetHTMLTemplate(t)

	// common api
	r.GET("/", page.GetIndexPage)
	r.GET("/page/groupPage", page.GetGroupPage)
	r.GET("/page/tool", page.GetToolPage)
	r.GET("/page/shell", page.GetShellPage)
	r.GET("/page/file", page.GetFilePage)
	r.GET("/page/browse", page.GetFileBrowsePage)
	r.GET("/page/ssh", page.GetSshPage)

	// websocket
	r.GET("/ws/index", page.GetWebsocketIndex)
	r.GET("/ws/ssh/:id", page.GetWebsocketSsh)

	// tools
	r.GET("/tools/cmd", v1.RunCmd)
	r.POST("/tools/upload", v1.FileUpload)
	r.GET("/tools/browse", v1.GetPathInfo)
	r.GET("/tools/download", v1.DownLoadFile)
	r.POST("/tools/delete", v1.DeleteFile)
	r.GET("/tools/export", v1.ExportData)
	r.POST("/tools/import", v1.ImportData)
	r.POST("/tools/upload_file", v1.FileUploadUnBlock)

	// restapi
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/host", v1.GetHosts)
		apiV1.GET("/host/:id", v1.GetOneHost)
		apiV1.POST("/host", v1.PostHost)
		apiV1.PUT("/host", v1.PutHost)
		apiV1.DELETE("/host/:id", v1.DeleteHost)

		apiV1.GET("/group", v1.GetGroups)
		apiV1.GET("/group/:id", v1.GetOneGroup)
		apiV1.POST("/group", v1.PostGroup)
		apiV1.PUT("/group", v1.PutGroup)
		apiV1.DELETE("/group/:id", v1.DeleteGroup)

		apiV1.GET("/tag", v1.GetTags)
		apiV1.GET("/tag/:id", v1.GetOneTag)
		apiV1.POST("/tag", v1.PostTag)
		apiV1.PUT("/tag", v1.PutTag)
		apiV1.DELETE("/tag/:id", v1.DeleteTag)

		apiV1.GET("/tunnel", v1.GetTunnels)
		apiV1.GET("/tunnel/:id", v1.GetOneTunnel)
		apiV1.POST("/tunnel", v1.PostTunnel)
		apiV1.PUT("/tunnel", v1.PutTunnel)
		apiV1.DELETE("/tunnel/:id", v1.DeleteTunnel)

		apiV1.GET("/job", v1.GetJobs)
		apiV1.GET("/job/:id", v1.GetOneJob)
		apiV1.POST("/job", v1.PostJob)
		apiV1.PUT("/job", v1.PutJob)
		apiV1.DELETE("/job/:id", v1.DeleteJob)
		apiV1.POST("/job/start", v1.StartJob)
		apiV1.POST("/job/stop", v1.StopJob)
	}

	addr := fmt.Sprintf("%s:%d", conf.DefaultConf.AppConf.HttpAddr, conf.DefaultConf.AppConf.HttpPort)
	log.Infof("Listening and serving HTTP on %s", addr)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}

func loadTemplate() (*template.Template, error) {
	box := packr.NewBox("../views")
	t := template.New("")
	if err := box.Walk(
		func(name string, file packd.File) error {
			h, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}
			t, err = t.New(name).Parse(string(h))
			if err != nil {
				return err
			}
			return nil
		},
	); err != nil {
		return nil, err
	}
	return t, nil
}
