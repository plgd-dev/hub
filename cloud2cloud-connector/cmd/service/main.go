package main

import (
	"context"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/service"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func main() {
	var cfg service.Config
	if err := config.LoadAndValidateConfig(&cfg); err != nil {
		log.Fatalf("cannot load config: %w", err)
	}
	logger, err := log.NewLogger(cfg.Log)
	if err != nil {
		log.Fatalf("cannot create logger: %w", err)
	}
	log.Set(logger)
	log.Infof("config: %v", cfg.String())
	s, err := service.New(context.Background(), cfg, logger)
	if err != nil {
		log.Fatalf("cannot create service: %w", err)
	}
	if err = s.Serve(); err != nil {
		log.Fatalf("cannot serve service: %v", err)
	}
}
