package transport

import (
	"github.com/prometheus/client_golang/prometheus"
	"oms/pkg/cache"
	"oms/pkg/utils"
	"time"
)

/*
Global data
*/

var (
	SSHClientPoll *cache.Cache
	CurrentFiles  *utils.SafeMap

	sshClientNum = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ssh_client_cache_nums",
		Help: "SSH Client Num in Lru Cache.",
	})
	fileListNum = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sftp_tran_file_nums",
		Help: "Current Files of Transport.",
	}, []string{"file"})
	currentSessionNum = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "current_session_nums",
		Help: "Current Nums of SSH Session.",
	})
)

func init() {
	CurrentFiles = utils.NewSageMap()
	SSHClientPoll = cache.NewCache(1000)

	prometheus.MustRegister(sshClientNum)
	prometheus.MustRegister(fileListNum)
	prometheus.MustRegister(currentSessionNum)

	go func() {
		for {
			<-time.After(time.Second * 5)
			sshClientNum.Set(float64(SSHClientPoll.Length()))

			CurrentFiles.Range(func(key, value interface{}) bool {
				fileListNum.WithLabelValues(key.(string)).Set(1)
				return true
			})
		}
	}()
}
