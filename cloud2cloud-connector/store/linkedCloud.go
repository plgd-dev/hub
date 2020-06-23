package store

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/go-ocf/cloud/authorization/oauth"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/kit/net/http/transport"
	"golang.org/x/oauth2"
)

type Events struct {
	Devices  []events.EventType `json:"Devices"`
	Device   []events.EventType `json:"Device"`
	Resource []events.EventType `json:"Resource"`
}

func makeMap(evs ...events.EventType) map[events.EventType]bool {
	m := make(map[events.EventType]bool)
	for _, e := range evs {
		m[e] = true
	}
	return m
}

func (e Events) NeedPullDevices() bool {
	set := makeMap(events.AllDevicesSubscriptions...)
	for _, v := range e.Devices {
		delete(set, v)
	}
	if len(set) != 0 {
		return true
	}
	return false
}

func (e Events) NeedPullDevice() bool {
	set := makeMap(events.AllDeviceSubscriptions...)
	for _, v := range e.Device {
		delete(set, v)
	}
	if len(set) != 0 {
		return true
	}
	return false
}

func (e Events) NeedPullResources() bool {
	set := makeMap(events.AllResourceSubscriptions...)
	for _, v := range e.Resource {
		delete(set, v)
	}
	if len(set) != 0 {
		return true
	}
	return false
}

type LinkedCloud struct {
	ID                           string       `json:"Id" bson:"_id"`
	Name                         string       `json:"Name"`
	OAuth                        oauth.Config `json:"OAuth"`
	RootCA                       []string     `json:"RootCA"`
	InsecureSkipVerify           bool         `json:"InsecureSkipVerify"`
	SupportedSubscriptionsEvents Events       `json:"SupportedSubscriptionEvents"`
}

func (l LinkedCloud) GetHTTPClient() *http.Client {
	pool := x509.NewCertPool()
	for _, ca := range l.RootCA {
		pool.AppendCertsFromPEM([]byte(ca))
	}
	t := transport.NewDefaultTransport()
	t.TLSClientConfig = &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: l.InsecureSkipVerify,
	}
	return &http.Client{
		Transport: t,
	}
}

func (l LinkedCloud) CtxWithHTTPClient(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, l.GetHTTPClient())
}
