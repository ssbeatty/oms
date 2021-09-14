package routers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"html/template"
	"io/ioutil"
	"oms/conf"
	v1 "oms/routers/api/v1"
	"oms/routers/page"
)

func InitGinServer() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	box := packr.NewBox("../static")
	r.StaticFS("/static", box)

	t, _ := loadTemplate()
	r.SetHTMLTemplate(t)

	r.GET("/", page.GetIndexPage)
	r.GET("/groupPage", page.GetGroupPage)
	r.GET("/tool", page.GetToolPage)
	r.GET("/shell", page.GetShellPage)
	r.GET("/shell_ws", page.GetShellWsPage)
	r.GET("/file", page.GetFilePage)
	r.GET("/browse", page.GetFileBrowsePage)
	r.GET("/ssh", page.GetSshPage)

	// websocket
	r.GET("ws/index", page.GetWebsocketIndex)
	r.GET("/ws/ssh/:id", page.GetWebsocketSsh)
	r.GET("/ws/shell", page.GetWebSocketShell)

	// tools
	r.GET("/tools/cmd", page.RunCmd)
	r.POST("/tools/upload", page.FileUpload)
	r.GET("/tools/browse", page.GetPathInfo)
	r.GET("/tools/download", page.DownLoadFile)
	r.POST("/tools/delete", page.DeleteFile)
	r.GET("/tools/export", page.ExportData)
	r.POST("/tools/import", page.ImportData)

	// @TODO api v1 开发前端时解决
	apiV1 := r.Group("/")
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
	}

	addr := fmt.Sprintf("%s:%d", conf.DefaultConf.AppConf.HttpAddr, conf.DefaultConf.AppConf.HttpPort)
	r.Run(addr)
}

// @TODO same name template 开发前端时解决
func loadTemplate() (*template.Template, error) {
	box := packr.NewBox("../views")
	t := template.New("")
	if err := box.Walk(
		func(name string, file packd.File) error {
			h, err := ioutil.ReadAll(file)
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
