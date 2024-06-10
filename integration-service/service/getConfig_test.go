package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"crypto/tls"
	"net/http"

	//serviceHTTP "github.com/plgd-dev/hub/v2/integration-service/service/http"
	"github.com/plgd-dev/hub/v2/integration-service/store"
	"github.com/plgd-dev/hub/v2/integration-service/test"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetConfig(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	storeDB, closeStore := test.NewStore(t)
	defer closeStore()

	for ind := 0; ind < 10; ind++ {

		strInd := strconv.Itoa(ind)
		r := &store.ConfigurationRecord{
			Id:    strInd,
			Name:  "Name" + strInd,
			Owner: "Boss" + strInd,
		}

		storeDB.CreateRecord(context.Background(), r)
	}

	cfg := test.MakeConfig(t)
	cfg.Clients.Storage.ExtendCronParserBySeconds = true
	cfg.Clients.Storage.CleanUpRecords = "" //"*/1 * * * * *"

	shutDown := test.SetUp(t, cfg)
	defer shutDown()

	configID := "5"
	url := "https://" + cfg.APIs.HTTP.Addr + "/api/v1/configuration/" + configID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Print("GET request error")
	}

	authToken := oauthTest.GetDefaultAccessToken(t)

	req.Header.Set("Authorization", "Bearer "+authToken)

	trans := http.DefaultTransport.(*http.Transport).Clone()
	trans.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: trans,
	}

	resp, _ := client.Do(req)

	jsonBytes, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}

	err = json.Unmarshal([]byte(jsonBytes), &result)

	if err != nil {
		fmt.Print("json parser error")
	}

	var id string

	if res, ok := result["result"]; ok {
		config := res.(map[string]interface{})

		if res, ok := config["id"]; ok {
			if strValue, ok := res.(string); ok {
				id = strValue
			}
		}
	}

	require.Equal(t, configID, id)

}
