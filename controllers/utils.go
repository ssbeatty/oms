package controllers

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"oms/logger"
	"os"
	"time"
)

func getFileName() string {
	var filePath string
	uploadDir := "static/upload/"
	err := os.MkdirAll(uploadDir, 744)
	if err != nil {
		logger.Logger.Println(err)
	}
	rand.Seed(time.Now().UnixNano())
	randNum := fmt.Sprintf("%d", rand.Intn(9999)+1000)
	hashName := md5.Sum([]byte(time.Now().Format("2006_01_02_15_04_05_") + randNum))
	fileName := fmt.Sprintf("%x", hashName)
	filePath = uploadDir + fileName
	return filePath
}
