package store

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"golang.org/x/oauth2"
)

type Events struct {
	Devices            []events.EventType `json:"devices"`
	Device             []events.EventType `json:"device"`
	Resource           []events.EventType `json:"resource"`
	StaticDeviceEvents bool               `json:"staticDeviceEvents"`
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
		return !e.StaticDeviceEvents
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
	URL                string   `json:"url"`
	RootCAs            []string `json:"rootCas"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
	UseSystemCAs       bool     `json:"useSystemCas"`
}

// https://github.com/plgd-dev/hub/blob/main/cloud2cloud-connector/swagger.yaml#/components/schemas/LinkedCloud
type LinkedCloud struct {
	ID                          string       `json:"id" bson:"_id"`
	Name                        string       `json:"name"`
	OAuth                       oauth.Config `json:"oauth"`
	SupportedSubscriptionEvents Events       `json:"supportedSubscriptionEvents"`
	Endpoint                    Endpoint     `json:"endpoint"`
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
