package main

import (
	"github.com/plgd-dev/cloud/coap-gateway/refImpl"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"os"
)

func main() {
	/*var config refImpl.Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("cannot parse configuration: %v", err)
	}*/

	var cfg refImpl.Config
	err := config.Load(&cfg)
	if err != nil {
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

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}