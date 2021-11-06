package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/server"
	"oms/pkg/logger"
)

func main() {
	// flags & init conf
	var configPath *string
	configPath = flag.String("config", "", "path of config")
	flag.Parse()

	conf, err := config.NewServerConfig(*configPath)
	if err != nil {
		panic(err)
	}

	// init db
	db := conf.Db
	models.InitModels(db.Dsn, db.DbName, db.UserName, db.PassWord, db.Driver)

	if conf.App.Mode == "dev" {
		logger.SetLevelAndFormat(logger.DebugLevel, &log.TextFormatter{})
	} else {
		logger.SetLevelAndFormat(logger.InfoLevel, &log.TextFormatter{})
	}

	srv := server.NewServer(conf)
	srv.Run()
}
