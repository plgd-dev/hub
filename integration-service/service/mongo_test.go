package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/hub/v2/integration-service/store"
	"github.com/plgd-dev/hub/v2/integration-service/test"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
)

func TestMongoDB(t *testing.T) {

	cfg := test.MakeConfig(t)
	cfg.Clients.Storage.ExtendCronParserBySeconds = true
	cfg.Clients.Storage.CleanUpRecords = "*/1 * * * * *"

	fmt.Printf("%v\n\n", test.MakeConfig(t))

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	shutDown := test.SetUp(t, cfg)
	defer shutDown()

	storeDB, closeStore := test.NewStore(t)
	defer closeStore()

	r := &store.ConfigurationRecord{
		Id:      "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
		Version: 0,
		Name:    "blankName",
		Owner:   "ja",
	}

	storeDB.CreateRecord(context.Background(), r)

}
