package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	rdService "github.com/plgd-dev/hub/v2/resource-directory/service"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test/service"
)

type testingT struct{}

func (testingT) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (testingT) FailNow() {
	fmt.Printf("FailNow\n")
	os.Exit(1)
}

func main() {
	services := flag.Int("services", 0, "enumerate services")
	rdConfigStr := flag.String("rdConfig", "", "path to resource directory config")
	flag.Parse()
	ctx := context.Background()
	if *services == 0 {
		*services = math.MaxInt
	}
	var rdConfig rdService.Config
	if *rdConfigStr == "" {
		rdConfig = rdTest.MakeConfig(testingT{})
	} else {
		err := config.Read(*rdConfigStr, &rdConfig)
		if err != nil {
			log.Fatalf("cannot unmarshal rdConfig: %v", err)
		}
		err = rdConfig.Validate()
		if err != nil {
			log.Fatalf("invalid config: %v", err)
		}
	}
	tearDown := service.SetUpServices(ctx, testingT{}, service.SetUpServicesConfig(*services), service.WithRDConfig(rdConfig))
	defer tearDown()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sigs
}
