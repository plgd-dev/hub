package main

import (
	"os"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/refImpl/maintenance"
)

func main() {
	if err := maintenance.PerformMaintenance(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
