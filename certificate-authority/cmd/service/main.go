package main

import (
	"context"

	"github.com/plgd-dev/hub/v2/certificate-authority/service"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func main() {
	var cfg service.Config
	err := config.LoadAndValidateConfig(&cfg)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	logger := log.NewLogger(cfg.Log)
	log.Set(logger)
	log.Infof("config: %v", cfg.String())

	fileWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("cannot create file fileWatcher: %v", err)
	}
	defer func() {
		_ = fileWatcher.Close()
	}()

	s, err := service.New(context.Background(), cfg, fileWatcher, logger)
	if err != nil {
		log.Fatalf("cannot create service: %v", err)
	}
	err = s.Serve()
	if err != nil {
		log.Fatalf("cannot serve service: %v", err)
	}
}
