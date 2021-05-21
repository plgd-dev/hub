package main

import (
	"os"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/maintenance"
)

func main() {
	if err := maintenance.PerformMaintenance(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
