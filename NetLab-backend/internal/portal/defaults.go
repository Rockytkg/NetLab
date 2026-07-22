package portal

import "time"

const (
	// NASPort is the standard Portal UDP port on access devices.
	NASPort = 2000

	defaultUDPWorkers  = 8
	queueWorkersFactor = 2
	protocolTimeout    = 5 * time.Second
	shutdownTimeout    = 5 * time.Second
)
