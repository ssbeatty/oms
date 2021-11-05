package utils

import (
	"fmt"
	"math/big"
	"net"
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
