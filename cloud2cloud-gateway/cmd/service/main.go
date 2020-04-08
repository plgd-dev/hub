package main

import (
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/refImpl"
	"github.com/kelseyhightower/envconfig"
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
	}
}
