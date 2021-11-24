package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ParseUploadPath(remote string, filename string) (string, string) {
	var remoteFile, remoteDir string
	if remote != "" {
		if remote[len(remote)-1] == '/' {
			remoteFile = filepath.ToSlash(filepath.Join(remote, filepath.Base(filename)))
			remoteDir = remote
		} else {
			remoteFile = remote
			remoteDir = filepath.ToSlash(filepath.Dir(remoteFile))
		}
	} else {
		remoteFile = filename
		remoteDir = filepath.ToSlash(filepath.Dir(remoteFile))
	}
	return remoteFile, remoteDir
}

func GetFileExt(path string) string {
	args := strings.Split(path, ".")
	if len(args) < 2 {
		return ""
	} else if len(args) == 2 && args[0] == "" {
		return ""
	}

	// 特殊的后缀
	if strings.HasSuffix(path, ".tar.gz") {
		return ".tar.gz"
	}
	return args[len(args)-1]
}
