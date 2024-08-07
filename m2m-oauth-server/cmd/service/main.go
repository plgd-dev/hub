package main

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/service"
	"github.com/plgd-dev/hub/v2/pkg/build"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
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
	logger.Debugf("version: %v, buildDate: %v, buildRevision %v", build.Version, build.BuildDate, build.CommitHash)
	logger.Infof("config: %v", cfg.String())

	if err := run(cfg, logger); err != nil {
		log.Fatalf("cannot run service: %v", err)
	}
}
