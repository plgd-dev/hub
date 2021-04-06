package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/resource-aggregate/refImpl"
	"github.com/plgd-dev/kit/log"
)

func main() {
	var config refImpl.Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}
	if server, err := refImpl.Init(config); err != nil {
		log.Fatalf("cannot init server: %v", err)
	} else {
		if err = server.Serve(); err != nil {
			log.Fatalf("unexpected ends: %v", err)
		}
		server.Shutdown()
	}
}
