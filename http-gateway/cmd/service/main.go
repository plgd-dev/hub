package main

import (
	"context"

	"github.com/plgd-dev/hub/http-gateway/service"
	"github.com/plgd-dev/hub/pkg/config"
	"github.com/plgd-dev/hub/pkg/log"
)

func main() {
	var cfg service.Config
	err := config.LoadAndValidateConfig(&cfg)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	logger, err := log.NewLogger(cfg.Log)
	if err != nil {
		log.Fatalf("cannot create logger: %v", err)
	}
	log.Set(logger)
	log.Infof("config: %v", cfg.String())
	s, err := service.New(context.Background(), cfg, logger)
	if err != nil {
		log.Fatalf("cannot create service: %v", err)
	}
	err = s.Serve()
	if err != nil {
		log.Fatalf("cannot serve service: %v", err)
	}
}
