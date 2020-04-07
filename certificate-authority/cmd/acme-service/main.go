package main

import (
	"github.com/go-ocf/ocf-cloud/certificate-authority/acme/refImpl"
	"github.com/go-ocf/kit/log"
	"github.com/kelseyhightower/envconfig"
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
