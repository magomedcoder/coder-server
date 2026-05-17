package mcpclient

import "sync/atomic"

var maxTrackedCallStatServerIDs atomic.Int64
var callToolStatDistinctCount atomic.Uint64

func SetMaxTrackedCallStatServerIDs(n int) {
	if n < 0 {
		n = 0
	}

	maxTrackedCallStatServerIDs.Store(int64(n))
}
