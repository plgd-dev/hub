package main

import (
	"log"

	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/kit/config"
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
