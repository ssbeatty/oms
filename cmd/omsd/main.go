package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/server"
	"oms/pkg/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// flags & init conf

	configPath := flag.String("config", "", "path of config")
	flag.Parse()

	conf, err := config.NewServerConfig(*configPath)
	if err != nil {
		panic(err)
	}

	// init db
	db := conf.Db
	if err := models.InitModels(db.Dsn, db.DbName, db.UserName, db.PassWord, db.Driver); err != nil {
		panic(fmt.Sprintf("init db error: %v", err))
	}

	if conf.App.Mode == "dev" {
		logger.SetLevelAndFormat(logger.DebugLevel, &log.TextFormatter{})
	} else {
		logger.SetLevelAndFormat(logger.InfoLevel, &log.TextFormatter{})
	}

	// run server
	srv := server.NewServer(conf)
	srv.Run()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Info("程序退出")
}
