package test

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/test/config"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type EventsServer struct {
	uri      string
	listener net.Listener
	cleanUp  func()
}

type Event struct {
	header events.EventHeader
	data   interface{}
}

func (e *Event) GetHeader() events.EventHeader {
	return e.header
}

func (e *Event) GetData() interface{} {
	return e.data
}

type EventChan = chan Event

func WaitForEvents(ch EventChan, timeout time.Duration) []Event {
	var events []Event
	stop := false
	for !stop {
		select {
		case ev := <-ch:
			events = append(events, ev)
		case <-time.After(timeout):
			stop = true
		}
	}
	return events
}

func DecodeEvent(t *testing.T, etype events.EventType, data []byte) interface{} {
	switch etype {
	case events.EventType_ResourcesPublished:
		fallthrough
	case events.EventType_ResourcesUnpublished:
		var links schema.ResourceLinks
		err := json.Decode(data, &links)
		assert.NoError(t, err)
		return links
	}

	return nil
}

func NewEventsServer(t *testing.T, uri string) *EventsServer {
	loggerCfg := log.Config{Debug: true}
	logger, err := log.NewLogger(loggerCfg)
	require.NoError(t, err)

	listenCfg := config.MakeListenerConfig("localhost:")
	listenCfg.TLS.ClientCertificateRequired = false

	certManager, err := server.New(listenCfg.TLS, logger)
	require.NoError(t, err)

	listener, err := tls.Listen("tcp", listenCfg.Addr, certManager.GetTLSConfig())
	require.NoError(t, err)

	return &EventsServer{
		uri:      uri,
		listener: listener,
		cleanUp: func() {
			certManager.Close()
		},
	}
}

func (s *EventsServer) GetPort(t *testing.T) string {
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	require.NoError(t, err)
	return port
}

func (s *EventsServer) Run(t *testing.T) EventChan {
	dataChan := make(EventChan, 1)
	go func() {
		r := router.NewRouter()
		r.StrictSlash(true)
		r.HandleFunc(s.uri, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, err := events.ParseEventHeader(r)
			assert.NoError(t, err)
			defer func() {
				_ = r.Body.Close()
			}()
			buf, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)

			data := DecodeEvent(t, h.EventType, buf)
			dataChan <- Event{
				header: h,
				data:   data,
			}
			w.WriteHeader(http.StatusOK)
		})).Methods("POST")
		_ = http.Serve(s.listener, r)
	}()
	return dataChan
}

func (s *EventsServer) Close(t *testing.T) {
	err := s.listener.Close()
	require.NoError(t, err)
	s.cleanUp()
}
