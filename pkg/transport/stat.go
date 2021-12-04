/*
rtop - the remote system monitoring utility
Copyright (c) 2015 RapidLoop
*/

package transport

import (
	"bufio"
	"bytes"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CmiTimeLayout = "20060102150405.999999"
)

type FSInfo struct {
	MountPoint string `json:"mount_point"`
	Used       uint64 `json:"used"`
	Free       uint64 `json:"free"`
}

type NetIntfInfo struct {
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
	Rx   uint64 `json:"rx"`
	Tx   uint64 `json:"tx"`
}

type cpuRaw struct {
	User    uint64 // time spent in user mode
	Nice    uint64 // time spent in user mode with low priority (nice)
	System  uint64 // time spent in system mode
	Idle    uint64 // time spent in the idle task
	Iowait  uint64 // time spent waiting for I/O to complete (since Linux 2.5.41)
	Irq     uint64 // time spent servicing  interrupts  (since  2.6.0-test4)
	SoftIrq uint64 // time spent servicing softirqs (since 2.6.0-test4)
	Steal   uint64 // time spent in other OSes when running in a virtualized environment
	Guest   uint64 // time spent running a virtual CPU for guest operating systems under the control of the Linux kernel.
	Total   uint64 // total of all time fields
}

type CPUInfo struct {
	Usage   float32 `json:"usage"`
	User    float32 `json:"user"`
	Nice    float32 `json:"nice"`
	System  float32 `json:"system"`
	Idle    float32 `json:"idle"`
	Iowait  float32 `json:"iowait"`
	Irq     float32 `json:"irq"`
	SoftIrq float32 `json:"soft_irq"`
	Steal   float32 `json:"steal"`
	Guest   float32 `json:"guest"`
}

type Stats struct {
	Uptime       time.Duration          `json:"uptime"`
	Hostname     string                 `json:"hostname"`
	Load1        string                 `json:"load_1"`
	Load5        string                 `json:"load_5"`
	Load10       string                 `json:"load_10"`
	RunningProcs string                 `json:"running_procs"`
	TotalProcs   string                 `json:"total_procs"`
	MemTotal     uint64                 `json:"mem_total"`
	MemFree      uint64                 `json:"mem_free"`
	MemBuffers   uint64                 `json:"mem_buffers"`
	MemCached    uint64                 `json:"mem_cached"`
	SwapTotal    uint64                 `json:"swap_total"`
	SwapFree     uint64                 `json:"swap_free"`
	FSInfos      []FSInfo               `json:"fs_infos"`
	NetIntf      map[string]NetIntfInfo `json:"-"`
	CPU          CPUInfo                `json:"cpu"`
	PreCPU       *cpuRaw                `json:"-"`
}

func NewStatus() *Stats {
	return &Stats{
		PreCPU: &cpuRaw{},
	}
}

func GetAllStats(client *Client, stats *Stats, wg *sync.WaitGroup) {
	getUptime(client, stats)
	getHostname(client, stats)
	getLoad(client, stats)
	getMemInfo(client, stats)
	getFSInfo(client, stats)
	getCPU(client, stats)
	if wg != nil {
		wg.Done()
	}
}

func runCommand(client *Client, command string) (stdout string, err error) {
	session, err := client.NewSession()
	if err != nil {
		//log.Print(err)
		return
	}
	defer session.Close()

	var buf bytes.Buffer
	session.SetStdout(&buf)
	err = session.Run(command)
	if err != nil {
		//log.Print(err)
		return
	}
	stdout = string(buf.Bytes())

	return
}

func winGetValRaw(client *Client, command string) (string, error) {
	valRaw, err := runCommand(client, command)
	if err != nil {
		return "", err
	}
	parts := strings.Fields(valRaw)
	if len(parts) == 2 {
		return parts[1], nil
	}

	return "", errors.New("command got error returns")
}

func getUptime(client *Client, stats *Stats) error {
	switch client.GetTargetMachineOs() {
	case GOOSLinux:
		uptime, err := runCommand(client, "/bin/cat /proc/uptime")
		if err != nil {
			return err
		}

		parts := strings.Fields(uptime)
		if len(parts) == 2 {
			var upsecs float64
			upsecs, err = strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return err
			}
			stats.Uptime = time.Duration(upsecs * 1e9)
		}
	case GOOSWindows:
		uptime, err := runCommand(client, "wmic os get lastbootuptime")
		if err != nil {
			return err
		}
		parts := strings.Fields(uptime)
		if len(parts) == 2 {
			var args []string
			cmiTimeStamp := parts[1]
			if strings.Contains(cmiTimeStamp, "+") {
				args = strings.Split(cmiTimeStamp, "+")
			} else if strings.Contains(cmiTimeStamp, "-") {
				args = strings.Split(cmiTimeStamp, "-")
			} else {
				return errors.New("got an error time format")
			}
			if len(args) == 2 {
				minutes, err := strconv.Atoi(args[1])
				if err != nil {
					return err
				}
				cstZone := time.FixedZone("GMT", minutes*60)
				parse, err := time.ParseInLocation(CmiTimeLayout, args[0], cstZone)
				if err != nil {
					return err
				}
				stats.Uptime = time.Now().In(cstZone).Sub(parse)
			}
		}
	}

	return nil
}

func getHostname(client *Client, stats *Stats) error {
	switch client.GetTargetMachineOs() {
	case GOOSLinux:
		hostname, err := runCommand(client, "/bin/hostname -f")
		if err != nil {
			return err
		}
		stats.Hostname = strings.TrimSpace(hostname)
	case GOOSWindows:
		hostname, err := runCommand(client, "wmic os get CSName")
		if err != nil {
			return err
		}
		parts := strings.Fields(hostname)
		if len(parts) == 2 {
			stats.Hostname = strings.TrimSpace(parts[1])
		}
	}
	return nil
}

func getLoad(client *Client, stats *Stats) error {
	switch client.GetTargetMachineOs() {
	case GOOSLinux:
		line, err := runCommand(client, "/bin/cat /proc/loadavg")
		if err != nil {
			return err
		}

		parts := strings.Fields(line)
		if len(parts) == 5 {
			stats.Load1 = parts[0]
			stats.Load5 = parts[1]
			stats.Load10 = parts[2]
			if i := strings.Index(parts[3], "/"); i != -1 {
				stats.RunningProcs = parts[3][0:i]
				if i+1 < len(parts[3]) {
					stats.TotalProcs = parts[3][i+1:]
				}
			}
		}
	case GOOSWindows:
		processNums, err := winGetValRaw(client, "wmic os get NumberOfProcesses")
		if err != nil {
			return err
		} else {
			stats.TotalProcs = processNums
		}
	}

	return nil
}

func getMemInfo(client *Client, stats *Stats) error {
	switch client.GetTargetMachineOs() {
	case GOOSLinux:
		lines, err := runCommand(client, "/bin/cat /proc/meminfo")
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(strings.NewReader(lines))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) == 3 {
				val, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					continue
				}
				val *= 1024
				switch parts[0] {
				case "MemTotal:":
					stats.MemTotal = val
				case "MemFree:":
					stats.MemFree = val
				case "Buffers:":
					stats.MemBuffers = val
				case "Cached:":
					stats.MemCached = val
				case "SwapTotal:":
					stats.SwapTotal = val
				case "SwapFree:":
					stats.SwapFree = val
				}
			}
		}

	case GOOSWindows:
		totalMemory, err := winGetValRaw(client, "wmic ComputerSystem get TotalPhysicalMemory")
		if err != nil {
			return err
		} else {
			val, err := strconv.Atoi(totalMemory)
			if err != nil {
				return err
			}
			stats.MemTotal = uint64(val)
		}

		freeMemory, err := winGetValRaw(client, "wmic OS get FreePhysicalMemory")
		if err != nil {
			return err
		} else {
			val, err := strconv.Atoi(freeMemory)
			if err != nil {
				return err
			}
			stats.MemFree = uint64(val)
		}

		swapTotal, err := winGetValRaw(client, "wmic pagefile get AllocatedBaseSize")
		if err != nil {
			return err
		} else {
			val, err := strconv.Atoi(swapTotal)
			if err != nil {
				return err
			}
			stats.SwapTotal = uint64(val)
		}

		swapCurrent, err := winGetValRaw(client, "wmic pagefile get CurrentUsage")
		if err != nil {
			return err
		} else {
			val, err := strconv.Atoi(swapCurrent)
			if err != nil {
				return err
			}
			stats.SwapFree = stats.SwapTotal - uint64(val)
		}
	}

	return nil
}

func getFSInfo(client *Client, stats *Stats) error {
	stats.FSInfos = stats.FSInfos[:0]
	switch client.GetTargetMachineOs() {
	case GOOSLinux:
		lines, err := runCommand(client, "/bin/df -B1")
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(strings.NewReader(lines))
		flag := 0
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			n := len(parts)
			dev := n > 0 && strings.Index(parts[0], "/dev/") == 0
			if n == 1 && dev {
				flag = 1
			} else if (n == 5 && flag == 1) || (n == 6 && dev) {
				i := flag
				flag = 0
				used, err := strconv.ParseUint(parts[2-i], 10, 64)
				if err != nil {
					continue
				}
				free, err := strconv.ParseUint(parts[3-i], 10, 64)
				if err != nil {
					continue
				}
				stats.FSInfos = append(stats.FSInfos, FSInfo{
					parts[5-i], used, free,
				})
			}
		}

	case GOOSWindows:
		lines, err := runCommand(client, "wmic logicaldisk get caption,freespace,size")
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(strings.NewReader(lines))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) != 3 {
				continue
			}
			used, err := strconv.Atoi(parts[1])
			if err != nil {
				continue
			}
			free, err := strconv.Atoi(parts[1])
			if err != nil {
				continue
			}
			stats.FSInfos = append(stats.FSInfos, FSInfo{
				parts[0], uint64(used), uint64(free),
			})
		}
	}

	return nil
}

func getInterfaces(client *Client, stats *Stats) (err error) {
	var lines string
	lines, err = runCommand(client, "/bin/ip -o addr")
	if err != nil {
		// try /sbin/ip
		lines, err = runCommand(client, "/sbin/ip -o addr")
		if err != nil {
			return
		}
	}

	if stats.NetIntf == nil {
		stats.NetIntf = make(map[string]NetIntfInfo)
	}

	scanner := bufio.NewScanner(strings.NewReader(lines))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 4 && (parts[2] == "inet" || parts[2] == "inet6") {
			ipv4 := parts[2] == "inet"
			intfname := parts[1]
			if info, ok := stats.NetIntf[intfname]; ok {
				if ipv4 {
					info.IPv4 = parts[3]
				} else {
					info.IPv6 = parts[3]
				}
				stats.NetIntf[intfname] = info
			} else {
				info := NetIntfInfo{}
				if ipv4 {
					info.IPv4 = parts[3]
				} else {
					info.IPv6 = parts[3]
				}
				stats.NetIntf[intfname] = info
			}
		}
	}

	return
}

func getInterfaceInfo(client *Client, stats *Stats) (err error) {
	lines, err := runCommand(client, "/bin/cat /proc/net/dev")
	if err != nil {
		return
	}

	if stats.NetIntf == nil {
		return
	} // should have been here already

	scanner := bufio.NewScanner(strings.NewReader(lines))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 17 {
			intf := strings.TrimSpace(parts[0])
			intf = strings.TrimSuffix(intf, ":")
			if info, ok := stats.NetIntf[intf]; ok {
				rx, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					continue
				}
				tx, err := strconv.ParseUint(parts[9], 10, 64)
				if err != nil {
					continue
				}
				info.Rx = rx
				info.Tx = tx
				stats.NetIntf[intf] = info
			}
		}
	}

	return
}

func parseCPUFields(fields []string, stat *cpuRaw) {
	numFields := len(fields)
	for i := 1; i < numFields; i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}

		stat.Total += val
		switch i {
		case 1:
			stat.User = val
		case 2:
			stat.Nice = val
		case 3:
			stat.System = val
		case 4:
			stat.Idle = val
		case 5:
			stat.Iowait = val
		case 6:
			stat.Irq = val
		case 7:
			stat.SoftIrq = val
		case 8:
			stat.Steal = val
		case 9:
			stat.Guest = val
		}
	}
}

func getCPU(client *Client, stats *Stats) error {
	switch client.GetTargetMachineOs() {
	case GOOSLinux:
		lines, err := runCommand(client, "/bin/cat /proc/stat")
		if err != nil {
			return err
		}

		var (
			nowCPU                               cpuRaw
			total                                float32
			PrevIdle, Idle, PrevNonIdle, NonIdle uint64
			PrevTotal, Total, totald, idled      uint64
		)

		scanner := bufio.NewScanner(strings.NewReader(lines))
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == "cpu" { // changing here if want to get every cpu-core's stats
				parseCPUFields(fields, &nowCPU)
				break
			}
		}
		if stats.PreCPU.Total == 0 { // having no pre raw cpu data
			goto END
		}

		total = float32(nowCPU.Total - stats.PreCPU.Total)
		stats.CPU.User = float32(nowCPU.User-stats.PreCPU.User) / total * 100
		stats.CPU.Nice = float32(nowCPU.Nice-stats.PreCPU.Nice) / total * 100
		stats.CPU.System = float32(nowCPU.System-stats.PreCPU.System) / total * 100
		stats.CPU.Idle = float32(nowCPU.Idle-stats.PreCPU.Idle) / total * 100
		stats.CPU.Iowait = float32(nowCPU.Iowait-stats.PreCPU.Iowait) / total * 100
		stats.CPU.Irq = float32(nowCPU.Irq-stats.PreCPU.Irq) / total * 100
		stats.CPU.SoftIrq = float32(nowCPU.SoftIrq-stats.PreCPU.SoftIrq) / total * 100
		stats.CPU.Guest = float32(nowCPU.Guest-stats.PreCPU.Guest) / total * 100
		PrevIdle = stats.PreCPU.Idle + stats.PreCPU.Iowait
		Idle = nowCPU.Idle + nowCPU.Iowait
		PrevNonIdle = stats.PreCPU.User + stats.PreCPU.Nice + stats.PreCPU.System + stats.PreCPU.Irq + stats.PreCPU.SoftIrq + stats.PreCPU.Steal
		NonIdle = nowCPU.User + nowCPU.Nice + nowCPU.System + nowCPU.Irq + nowCPU.SoftIrq + nowCPU.Steal

		PrevTotal = PrevIdle + PrevNonIdle
		Total = Idle + NonIdle
		totald = Total - PrevTotal
		idled = Idle - PrevIdle
		stats.CPU.Usage = float32(totald-idled) / float32(totald)

	END:
		stats.PreCPU = &nowCPU
	case GOOSWindows:
		cpuUsage, err := winGetValRaw(client, "wmic cpu get loadpercentage")
		if err != nil {
			return err
		} else {
			val, err := strconv.ParseFloat(cpuUsage, 32)
			if err != nil {
				return err
			}
			stats.CPU.Usage = float32(val)
		}
	}

	return nil
}
