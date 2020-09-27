package utils

import (
	"math"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

func GetSeverStatus() map[string]interface{} {
	cpuPercet, _ := cpu.Percent(time.Second, true)
	var cpuAll float64
	for _, v := range cpuPercet {
		cpuAll += v
	}
	m := make(map[string]interface{})
	loads, _ := load.Avg()
	m["load1"] = loads.Load1
	m["load5"] = loads.Load5
	m["load15"] = loads.Load15
	m["cpu"] = math.Round(cpuAll / float64(len(cpuPercet)))
	swap, _ := mem.SwapMemory()
	m["swap_mem"] = math.Round(swap.UsedPercent)
	vir, _ := mem.VirtualMemory()
	m["virtual_mem"] = math.Round(vir.UsedPercent)
	conn, _ := net.ProtoCounters(nil)
	parts, _ := disk.Partitions(true)
	for _, part := range parts {
		diskInfo, _ := disk.Usage(part.Mountpoint)
		m["disk_used_"+diskInfo.Path] = diskInfo.UsedPercent
		// GB
		m["disk_free_"+diskInfo.Path] = diskInfo.Free /1024/1024/1024
	}
	io1, _ := net.IOCounters(false)
	time.Sleep(time.Millisecond * 500)
	io2, _ := net.IOCounters(false)
	if len(io2) > 0 && len(io1) > 0 {
		m["net_io_send"] = (io2[0].BytesSent - io1[0].BytesSent) * 2
		m["net_io_recv"] = (io2[0].BytesRecv - io1[0].BytesRecv) * 2
	}
	t := time.Now()
	m["sys_time"] = strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + ":" + strconv.Itoa(t.Second())

	for _, v := range conn {
		m[v.Protocol] = v.Stats["CurrEstab"]
	}
	ioStat, _ := disk.IOCounters()
	for _, v := range ioStat {
		m["read_count_"+v.Name] = v.ReadCount
		m["write_count_"+v.Name] = v.WriteCount
		//MB
		m["read_bytes_"+v.Name] = float64(v.ReadBytes)/1024/1024
		m["write_bytes_"+v.Name] = float64(v.WriteBytes)/1024/1024
	}
	return m
}
