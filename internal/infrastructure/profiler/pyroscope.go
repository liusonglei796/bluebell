package profiler

import (
	"bluebell/internal/config"
	"github.com/grafana/pyroscope-go"
)

// Init starts continuous profiling sessions using Grafana Pyroscope SDK
func Init(cfg *config.PyroscopeConfig) (func() error, error) {
	if cfg == nil || !cfg.Enabled {
		return func() error { return nil }, nil
	}

	session, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ServiceName,
		ServerAddress:   cfg.Endpoint,
		Logger:          pyroscope.StandardLogger,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return nil, err
	}

	return session.Stop, nil
}
