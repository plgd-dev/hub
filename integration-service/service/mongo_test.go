package service_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/plgd-dev/hub/v2/integration-service/store"
	"github.com/plgd-dev/hub/v2/integration-service/test"
)

func TestMongoDB(t *testing.T) {

	storeDB, closeStore := test.NewStore(t)
	defer closeStore()

	for ind := 0; ind < 10; ind++ {

		strInd := strconv.Itoa(ind)
		r := &store.ConfigurationRecord{
			Id:    strInd,
			Name:  "Name" + strInd,
			Owner: "Boss1" + strInd,
		}

		storeDB.CreateRecord(context.Background(), r)
	}

}
