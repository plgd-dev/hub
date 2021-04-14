package main

import (
	"os"

	"github.com/plgd-dev/cloud/resource-aggregate/maintenance"
	"github.com/plgd-dev/kit/log"
)

func main() {
	if err := maintenance.PerformMaintenance(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
