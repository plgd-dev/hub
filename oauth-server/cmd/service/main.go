package main

import (
	"github.com/plgd-dev/cloud/oauth-server/service"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
)

func main() {
	var cfg service.Config
	err := config.Load(&cfg)
	if err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}
	server, err := service.New(cfg)
	if err != nil {
		log.Fatalf("cannot init server: %v", err)
	}
	if err := server.Serve(); err != nil {
		log.Fatalf("unexpected ends: %v", err)
	}
}
