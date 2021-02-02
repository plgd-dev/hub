package main

import (
	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
)

func main() {
	var cfg service.Config
	err := config.Load(&cfg)
	if err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}
	log.Setup(cfg.Log)
	log.Info(cfg.String())

	if server, err := service.New(cfg); err != nil {
		log.Fatalf("cannot init server: %v", err)
	} else {
		if err = server.Serve(); err != nil {
			log.Fatalf("unexpected ends: %v", err)
		}
	}
}
