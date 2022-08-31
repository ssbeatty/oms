package utils

import (
	"os"
	"path/filepath"
	"strconv"
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

func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func GetEnvInt(key string, fallback int) int {
	ret := fallback
	value, exists := os.LookupEnv(key)
	if !exists {
		return ret
	}
	if t, err := strconv.Atoi(value); err != nil { //nolint:gosec
		return ret
	} else {
		ret = t
	}
	return ret
}

func GetEnvBool(key string, fallback bool) bool {
	ret := fallback
	value, exists := os.LookupEnv(key)
	if !exists {
		return ret
	}
	if t, err := strconv.ParseBool(value); err != nil {
		return ret
	} else {
		ret = t
	}
	return ret
}
