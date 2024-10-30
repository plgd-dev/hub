package test

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type EventsServer struct {
	uri      string
	listener net.Listener
	cleanUp  func()
	wg       sync.WaitGroup
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

func decodeEvent(etype events.EventType, data []byte) (interface{}, error) {
	switch etype {
	case events.EventType_ResourcesPublished:
		fallthrough
	case events.EventType_ResourcesUnpublished:
		var links schema.ResourceLinks
		err := json.Decode(data, &links)
		if err != nil {
			return nil, err
		}
		return links, nil
	case events.EventType_ResourceChanged:
		var colContent []map[interface{}]interface{}
		err := json.Decode(data, &colContent)
		if err == nil {
			return colContent, nil
		}
		var content map[interface{}]interface{}
		err = json.Decode(data, &content)
		if err != nil {
			return nil, err
		}
		return content, nil
	case events.EventType_DevicesRegistered:
		fallthrough
	case events.EventType_DevicesUnregistered:
		fallthrough
	case events.EventType_DevicesOnline:
		fallthrough
	case events.EventType_DevicesOffline:
		var devices []map[string]string
		err := json.Decode(data, &devices)
		if err != nil {
			return nil, err
		}
		return devices, nil
	}

	return nil, nil
}

func NewEventsServer(t *testing.T, uri string) *EventsServer {
	loggerCfg := log.MakeDefaultConfig()
	logger := log.NewLogger(loggerCfg)

	listenCfg := config.MakeListenerConfig("localhost:")
	listenCfg.TLS.ClientCertificateRequired = false

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	certManager, err := server.New(listenCfg.TLS, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)

	listener, err := tls.Listen("tcp", listenCfg.Addr, certManager.GetTLSConfig())
	require.NoError(t, err)

	return &EventsServer{
		uri:      uri,
		listener: listener,
		cleanUp: func() {
			certManager.Close()
			err = fileWatcher.Close()
			require.NoError(t, err)
		},
	}
}

func (s *EventsServer) GetPort(t *testing.T) string {
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	require.NoError(t, err)
	return port
}

func (s *EventsServer) Run(t *testing.T) EventChan {
	dataChan := make(EventChan, 8)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		r := router.NewRouter()
		r.StrictSlash(true)
		r.HandleFunc(s.uri, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, err := events.ParseEventHeader(r)
			assert.NoError(t, err)
			defer func() {
				_ = r.Body.Close()
			}()
			buf, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			data, err := decodeEvent(h.EventType, buf)
			assert.NoError(t, err)
			dataChan <- Event{
				header: h,
				data:   data,
			}
			w.WriteHeader(http.StatusOK)
		})).Methods(http.MethodPost)
		_ = http.Serve(s.listener, r)
	}()
	return dataChan
}

func (s *EventsServer) Close(t *testing.T) {
	err := s.listener.Close()
	require.NoError(t, err)
	s.cleanUp()
}

func (s *EventsServer) WaitForClose() {
	s.wg.Wait()
}
