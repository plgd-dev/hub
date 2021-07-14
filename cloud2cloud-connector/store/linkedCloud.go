package store

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"golang.org/x/oauth2"
)

type Events struct {
	Devices            []events.EventType `json:"Devices"`
	Device             []events.EventType `json:"Device"`
	Resource           []events.EventType `json:"Resource"`
	StaticDeviceEvents bool               `json:"StaticDeviceEvents"`
}

func makeMap(evs ...events.EventType) map[events.EventType]bool {
	m := make(map[events.EventType]bool)
	for _, e := range evs {
		m[e] = true
	}
	return m
}

func (e Events) NeedPullDevices() bool {
	set := makeMap(events.AllDevicesEvents...)
	for _, v := range e.Devices {
		delete(set, v)
	}
	return len(set) != 0
}

func (e Events) NeedPullDevice() bool {
	set := makeMap(events.AllDeviceEvents...)
	for _, v := range e.Device {
		delete(set, v)
	}
	if len(set) != 0 {
		return true && !e.StaticDeviceEvents
	}
	return false
}

func (e Events) NeedPullResources() bool {
	set := makeMap(events.AllResourceEvents...)
	for _, v := range e.Resource {
		delete(set, v)
	}
	return len(set) != 0
}

type Endpoint struct {
	URL                string   `json:"URL"`
	RootCAs            []string `json:"RootCAs"`
	InsecureSkipVerify bool     `json:"InsecureSkipVerify"`
	UseSystemCAs       bool     `json:"UseSystemCAs"`
}

type LinkedCloud struct {
	ID                           string       `json:"Id" bson:"_id"`
	Name                         string       `json:"Name"`
	OAuth                        oauth.Config `json:"OAuth"`
	SupportedSubscriptionsEvents Events       `json:"SupportedSubscriptionEvents"`
	Endpoint                     Endpoint     `json:"Endpoint"`
}

func (l LinkedCloud) GetHTTPClient() *http.Client {
	var pool *x509.CertPool
	if l.Endpoint.UseSystemCAs {
		pool, _ = x509.SystemCertPool()
	}
	if pool == nil {
		pool = x509.NewCertPool()
	}

	for _, ca := range l.Endpoint.RootCAs {
		pool.AppendCertsFromPEM([]byte(ca))
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: l.Endpoint.InsecureSkipVerify,
	}
	t.MaxIdleConns = 1
	t.MaxConnsPerHost = 1
	t.MaxIdleConnsPerHost = 1
	t.IdleConnTimeout = time.Second
	return &http.Client{
		Timeout:   time.Second * 10,
		Transport: t,
	}
}

func (l LinkedCloud) CtxWithHTTPClient(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, l.GetHTTPClient())
}
