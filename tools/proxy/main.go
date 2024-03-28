package main

import (
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/build"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func isErrNetClosing(err error) bool {
	return strings.Contains(err.Error(), "use of closed network connection")
}

func main() {
	var cfg Config
	err := config.LoadAndValidateConfig(&cfg)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	logger := log.NewLogger(cfg.Log)
	log.Set(logger)
	logger.Debugf("version: %v, buildDate: %v, buildRevision %v", build.Version, build.BuildDate, build.CommitHash)
	logger.Infof("config: %v", cfg.String())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	state, err := NewState(cfg.Clients.Storage.Directory)
	if err != nil {
		logger.Fatalf("cannot initialize state: %v", err)
	}
	tunnels := make([]*Tunnel, 0, len(cfg.Apis.TCP.Tunnels))
	for idx, tunnelConfig := range cfg.Apis.TCP.Tunnels {
		tunnel, err := NewTunnel(tunnelConfig, state, logger)
		if err != nil {
			logger.Fatalf("cannot create tunnel[%v]: %v", idx, err)
		}
		tunnels = append(tunnels, tunnel)
	}

	if cfg.Log.Level == log.DebugLevel {
		go func() {
			var m runtime.MemStats
			for {
				runtime.ReadMemStats(&m)
				log.Debugf("memstat Alloc = %v MiB, Sys = %v MiB, numGoroutines = %v", m.Alloc/1024/1024, m.Sys/1024/1024, runtime.NumGoroutine())
				for _, t := range tunnels {
					t.logger.Debugf("numConnections = %v", t.numConnections())
				}
				time.Sleep(5 * time.Second)
			}
		}()
	}

	var wg sync.WaitGroup
	wg.Add(len(tunnels))
	for _, tunnel := range tunnels {
		go func(tunnel *Tunnel) {
			defer wg.Done()
			err := tunnel.Serve()
			if err != nil && !isErrNetClosing(err) {
				logger.Errorf("tunnel serve failed: %v", err)
			}
		}(tunnel)
	}

	<-signals
	for _, tunnel := range tunnels {
		err := tunnel.Close()
		if err != nil {
			logger.Errorf("tunnel close failed: %v", err)
		}
	}
	wg.Wait()
}
