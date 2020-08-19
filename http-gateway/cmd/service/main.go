package main

import (
	"io/ioutil"
	"log"

	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/jessevdk/go-flags"
)

type Config struct {
	ConfigPath string `long:"config" env:"CONFIG" default:"httpgw.yaml" description:"yaml config file path"`
}

func main() {
	var config Config
	_, err := flags.Parse(&config)
	if err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}
	cfg, err := ioutil.ReadFile(config.ConfigPath)
	if err != nil {
		log.Fatalf("invalid config file path %s: %v", config.ConfigPath, err)
	}
	conf, err := service.ParseConfig(string(cfg))
	if err != nil {
		log.Fatalf("invalid parse config %s: %v", string(cfg), err)
	}
	server, err := service.New(conf)
	if err != nil {
		log.Fatalf("cannot init server: %v", err)
	}
	if err := server.Serve(); err != nil {
		log.Fatalf("unexpected ends: %v", err)
	}
}
