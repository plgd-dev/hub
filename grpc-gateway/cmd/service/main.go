package main

import (
	"github.com/plgd-dev/cloud/grpc-gateway/refImpl"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
)

func main() {
	var cfg refImpl.Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}
	if server, err := refImpl.Init(cfg); err != nil {
		log.Fatalf("cannot init server: %v", err)
	} else {
		if err = server.Serve(); err != nil {
			log.Fatalf("unexpected ends: %v", err)
		}
	}
}
