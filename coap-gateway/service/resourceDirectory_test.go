package service_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/coap-gateway/uri"
	testCfg "github.com/plgd-dev/hub/test/config"
	"github.com/stretchr/testify/require"
)

type TestResource struct {
	DeviceID      string   `json:"di,omitempty"`
	Href          string   `json:"href"`
	ID            string   `json:"id,omitempty"`
	Interfaces    []string `json:"if,omitempty"`
	InstanceID    uint64   `json:"-"`
	ResourceTypes []string `json:"rt,omitempty"`
	Type          []string `json:"type,omitempty"`
}

type TestWkRD struct {
	DeviceID         string         `json:"di"`
	Links            []TestResource `json:"links"`
	TimeToLive       int            `json:"ttl"`
	TimeToLiveLegacy int            `json:"lt"`
}

var tblResourceDirectory = []testEl{
	{"BadRequest0", input{coapCodes.POST, `{ "di":"` + CertIdentity + `" }`, nil}, output{coapCodes.BadRequest, `empty links`, nil}, true},
	{"BadRequest1", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":"abc" }`, nil}, output{coapCodes.BadRequest, `cbor: cannot unmarshal`, nil}, true},
	{"BadRequest2", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ "abc" ]}`, nil}, output{coapCodes.BadRequest, `cbor: cannot unmarshal`, nil}, true},
	{"BadRequest3", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ {} ]}`, nil}, output{coapCodes.BadRequest, `invalid TimeToLive`, nil}, true},
	{"BadRequest4", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "href":"" } ]}`, nil}, output{coapCodes.BadRequest, `invalid TimeToLive`, nil}, true},
	{"BadRequest5", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ {} ], "ttl":-1}`, nil}, output{coapCodes.BadRequest, `invalid TimeToLive`, nil}, true},
	{"BadRequest6", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ {} ], "lt":-1}`, nil}, output{coapCodes.BadRequest, `invalid TimeToLive`, nil}, true},
	{"BadRequest7", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"" } ], "ttl":12345}`, nil}, output{coapCodes.BadRequest, `invalid resource href`, nil}, true},
	{"BadRequest8", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "href":"" } ], "ttl":12345}`, nil}, output{coapCodes.BadRequest, `invalid resource href`, nil}, true},
	{"Changed0", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestAResourceHref + `" } ], "ttl":12345}`, nil},
		output{coapCodes.Changed, TestWkRD{
			DeviceID:         CertIdentity,
			TimeToLive:       12345,
			TimeToLiveLegacy: 12345,
			Links: []TestResource{
				{
					DeviceID: CertIdentity,
					Href:     TestAResourceHref,
				},
			},
		}, nil}, false},

	{"Changed1", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"/b" } ], "ttl":12345}`, nil},
		output{coapCodes.Changed, TestWkRD{
			DeviceID:         CertIdentity,
			TimeToLive:       12345,
			TimeToLiveLegacy: 12345,
			Links: []TestResource{
				{
					DeviceID: CertIdentity,
					Href:     "/b",
				},
			},
		}, nil}, false},
	{"Changed2", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"/b" } , { "di":"` + CertIdentity + `", "href":"/c" }], "ttl":12345}`, nil},
		output{coapCodes.Changed, TestWkRD{
			DeviceID:         CertIdentity,
			TimeToLive:       12345,
			TimeToLiveLegacy: 12345,
			Links: []TestResource{
				{
					DeviceID: CertIdentity,
					Href:     "/b",
				},
				{
					DeviceID: CertIdentity,
					Href:     "/c",
				},
			},
		}, nil}, false},
	{"Changed3", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestAResourceHref + `" } ], "ttl":0}`, nil},
		output{coapCodes.Changed, TestWkRD{
			DeviceID:         CertIdentity,
			TimeToLive:       0,
			TimeToLiveLegacy: 0,
			Links: []TestResource{
				{
					DeviceID: CertIdentity,
					Href:     TestAResourceHref,
				},
			},
		}, nil}, false},
	{"Changed4", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestAResourceHref + `" } ], "lt":0}`, nil},
		output{coapCodes.Changed, TestWkRD{
			DeviceID:         CertIdentity,
			TimeToLive:       0,
			TimeToLiveLegacy: 0,
			Links: []TestResource{
				{
					DeviceID: CertIdentity,
					Href:     TestAResourceHref,
				},
			},
		}, nil}, false},
}

func TestResourceDirectoryPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	testSignUpIn(t, CertIdentity, co)

	for _, test := range tblResourceDirectory {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.ResourceDirectory, test, co)
		}
		t.Run(test.name, tf)
	}
}

func TestResourceDirectoryDeleteHandler(t *testing.T) {
	//set counter 0, when other test run with this that it can be modified

	deletetblResourceDirectory := []testEl{
		{"NotExist1", input{coapCodes.DELETE, ``, []string{"di=c", "ins=5"}}, output{coapCodes.BadRequest, `cannot find observed resources using query`, nil}, true},                 // Non-existent device ID.
		{"NotExist2", input{coapCodes.DELETE, ``, []string{"ins=4"}}, output{coapCodes.BadRequest, `not found`, nil}, true},                                                          // Device ID empty.
		{"NotExist3", input{coapCodes.DELETE, ``, []string{`di=` + CertIdentity, "ins=999"}}, output{coapCodes.BadRequest, `cannot find observed resources using query`, nil}, true}, // Instance ID non-existent.
		{"Exist1", input{coapCodes.DELETE, ``, []string{`di=` + CertIdentity}}, output{coapCodes.Deleted, nil, nil}, false},                                                          // If instanceIDs empty, all instances for a given device ID should be unpublished.
		{"NotExist4", input{coapCodes.DELETE, ``, []string{`di=` + CertIdentity}}, output{coapCodes.BadRequest, `cannot find observed resources using query`, nil}, true},
	}

	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	testSignUpIn(t, CertIdentity, co)

	// Publish resources first!
	for _, test := range tblResourceDirectory {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.ResourceDirectory, test, co)
		}
		t.Run(test.name, tf)
	}

	//delete resources
	for _, test := range deletetblResourceDirectory {
		tf := func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := tcp.NewDeleteRequest(ctx, uri.ResourceDirectory)
			require.NoError(t, err)
			for _, q := range test.in.queries {
				req.AddOptionString(message.URIQuery, q)
			}
			resp, err := co.Do(req)
			if err != nil {
				t.Fatalf("Cannot send/retrieve msg: %v", err)
			}
			testValidateResp(t, test, resp)
		}
		t.Run(test.name, tf)
	}
}

type TestGetSelector struct {
	Selector uint64 `json:"sel"`
}

func TestResourceDirectoryGetSelector(t *testing.T) {
	tbl := []testEl{
		{"GetSelector", input{coapCodes.GET, ``, []string{}}, output{coapCodes.Content, TestGetSelector{}, nil}, false},
	}

	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	testSignUpIn(t, CertIdentity, co)

	for _, test := range tbl {
		tf := func(t *testing.T) {
			req, err := tcp.NewGetRequest(co.Context(), uri.ResourceDirectory)
			require.NoError(t, err)
			for _, q := range test.in.queries {
				req.AddOptionString(message.URIQuery, q)
			}
			resp, err := co.Do(req)
			require.NoError(t, err)
			testValidateResp(t, test, resp)
		}
		t.Run(test.name, tf)
	}
}
