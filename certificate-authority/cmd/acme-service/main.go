package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/certificate-authority/acme/refImpl"
	"github.com/plgd-dev/kit/log"
)

func main() {
	var config refImpl.Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}
	server, err := refImpl.Init(config)
	if err != nil {
		log.Fatalf("cannot init server: %v", err)
	}
	if err := server.Serve(); err != nil {
		log.Fatalf("unexpected ends: %v", err)
	}
}
