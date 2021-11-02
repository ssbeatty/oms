package main

import (
	log "github.com/sirupsen/logrus"
	_ "oms/conf"
	_ "oms/pkg/schedule"
	_ "oms/pkg/tunnel"
	"oms/routers"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {
	routers.InitGinServer()
}
