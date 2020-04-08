package main

import (
	"os"

	"github.com/go-ocf/cloud/resource-aggregate/refImpl/maintenance"
	"github.com/go-ocf/kit/log"
)

func main() {
	if err := maintenance.PerformMaintenance(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
