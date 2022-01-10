package main

import (
	"context"

	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func main() {
	var cfg service.Config
	err := config.LoadAndValidateConfig(&cfg)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	logger, err := log.NewLogger(cfg.Log.Embedded)
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
