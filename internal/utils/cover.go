package utils

import (
	"crypto/sha1"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
)

func InetNtoA(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		byte(ip>>56), byte(ip>>48), byte(ip>>40), byte(ip>>32), int32(ip))
}

func InetAtoN(ip string, port int) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()<<32 + int64(port)
}

// IntChangeToSize 转换单位
func IntChangeToSize(s int64) string {
	// 1k 以内
	if s < 1024 {
		return fmt.Sprintf("%.2fb", float64(s))
	} else if s < 1024*1024 {
		return fmt.Sprintf("%.2fkb", float64(s)/1024.0)
	} else if s < 1024*1024*1024 {
		return fmt.Sprintf("%.2fmb", float64(s)/1048576.0)
	} else {
		return fmt.Sprintf("%.2fgb", float64(s)/1073741824.0)
	}
}

func isPort(p string) bool {
	port, err := strconv.Atoi(p)
	if err != nil {
		return false
	}
	if port > 0 && port < 65535 {
		return true
	} else {
		return false
	}
}

// IsAddr 判断string是否为ip地址
// :9091 127.0.0.1:9090 0.0.0.0:9090
func IsAddr(address string) bool {
	if !strings.Contains(address, ":") {
		return false
	}
	args := strings.Split(address, ":")
	switch len(args) {
	case 0:
		return false
	case 1:
		return isPort(args[0])
	case 2:
		if args[0] == "" {
			args[0] = "0.0.0.0"
		}
		ip := net.ParseIP(args[0])
		if ip == nil {
			return false
		}
		return isPort(args[1])
	default:
		return false
	}
}

// HashSha1 sha1 加密
func HashSha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)

	return string(bs)
}
