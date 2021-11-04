package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"oms/conf"
	_ "oms/pkg/schedule"
	_ "oms/pkg/tunnel"
	"oms/routers"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	if conf.DefaultConf.AppConf.RunMode == "dev" {
		log.SetLevel(log.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		log.SetLevel(log.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
}

func main() {
	routers.InitGinServer()
}
