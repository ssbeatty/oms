package transport

import (
	"oms/pkg/cache"
	"sync"
)

/*
Global data
*/

var CurrentFiles *sync.Map
var SSHClientPoll *cache.Cache

func init() {
	CurrentFiles = &sync.Map{}
	SSHClientPoll = cache.NewCache(1000)
}
