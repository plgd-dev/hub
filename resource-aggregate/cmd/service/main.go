package main

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
)

func run(cfg service.Config, logger log.Logger) error {
	fileWatcher, err := fsnotify.NewWatcher(logger)
	if err != nil {
		return fmt.Errorf("cannot create file fileWatcher: %w", err)
	}
	defer func() {
		_ = fileWatcher.Close()
	}()

	s, err := service.New(context.Background(), cfg, fileWatcher, logger)
	if err != nil {
		return fmt.Errorf("cannot create service: %w", err)
	}
	err = s.Serve()
	if err != nil {
		return fmt.Errorf("cannot serve service: %w", err)
	}
	return nil
}

func main() {
	var cfg service.Config
	err := config.LoadAndValidateConfig(&cfg)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	logger := log.NewLogger(cfg.Log)
	log.Set(logger)
	log.Infof("config: %v", cfg.String())

	if err := run(cfg, logger); err != nil {
		log.Fatalf("cannot run service: %v", err)
	}
}
