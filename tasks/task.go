package tasks

import (
	"errors"
	"fmt"
	"io/ioutil"
	"oms/logger"
	"oms/models"
	"os"
	"path/filepath"
)

func ClearCache() error {
	uploadDir := "static/upload"
	logger.Logger.Println("======================Task ClearCache start======================")
	info, err := os.Stat(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Logger.Println("can't found upload dir, mkdir now")
			err = os.MkdirAll(uploadDir, os.ModePerm)
			return err
		}
	}
	if info.IsDir() {
		files, _ := ioutil.ReadDir(uploadDir)
		for _, f := range files {
			GetHostByKeyFile := fmt.Sprintf("%s/%s", uploadDir, f.Name())
			length := models.GetHostByKeyFile(GetHostByKeyFile)
			if length == 0 {
				localPath := filepath.ToSlash(filepath.Join(uploadDir, filepath.Base(f.Name())))
				err = os.Remove(localPath)
				logger.Logger.Println("remove: " + localPath)
				if err != nil {
					logger.Logger.Println(err)
				}
			}
		}
	} else {
		logger.Logger.Println("get upload dir error")
		return errors.New("get upload dir error")
	}
	logger.Logger.Println("======================Task ClearCache stop ======================")
	return nil
}
