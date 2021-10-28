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
