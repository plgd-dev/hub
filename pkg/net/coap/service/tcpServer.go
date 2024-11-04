package service

import (
	"fmt"

	coapDtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapTcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
	coapUdpServer "github.com/plgd-dev/go-coap/v3/udp/server"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"go.opentelemetry.io/otel/trace"
)

type tcpListener struct {
	coapTcpServer.Listener
	close func()
}

type tcpServer struct {
	coapServer *coapTcpServer.Server
	listener   *tcpListener
}

func (s *tcpServer) Serve() error {
	return s.coapServer.Serve(s.listener)
}

func (s *tcpServer) Close() error {
	s.coapServer.Stop()
	s.listener.close()
	return nil
}

func newTCPListener(config Config, serviceOpts Options, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*tcpListener, error) {
	if !config.TLS.IsEnabled() {
		listener, err := net.NewTCPListener("tcp", config.Addr)
		if err != nil {
			return nil, fmt.Errorf("cannot create tcp listener: %w", err)
		}
		closeListener := func() {
			if err := listener.Close(); err != nil {
				logger.Errorf("failed to close tcp listener: %v", err)
			}
		}
		return &tcpListener{
			Listener: listener,
			close:    closeListener,
		}, nil
	}

	var closeListener fn.FuncList
	coapsTLS, err := certManagerServer.New(config.TLS.Embedded, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create tls cert manager: %w", err)
	}
	closeListener.AddFunc(coapsTLS.Close)
	tlsCfg := coapsTLS.GetTLSConfig()
	if serviceOpts.OverrideTLSConfig != nil {
		tlsCfg = serviceOpts.OverrideTLSConfig(tlsCfg, coapsTLS.VerifyByCRL)
	}
	listener, err := net.NewTLSListener("tcp", config.Addr, tlsCfg)
	if err != nil {
		closeListener.Execute()
		return nil, fmt.Errorf("cannot create tcp-tls listener: %w", err)
	}
	closeListener.AddFunc(func() {
		if err := listener.Close(); err != nil {
			logger.Errorf("failed to close tcp-tls listener: %w", err)
		}
	})
	return &tcpListener{
		Listener: listener,
		close:    closeListener.ToFunction(),
	}, nil
}

func newTCPServer(config Config, serviceOpts Options, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...interface {
	coapTcpServer.Option
	coapDtlsServer.Option
	coapUdpServer.Option
},
) (*tcpServer, error) {
	listener, err := newTCPListener(config, serviceOpts, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create listener: %w", err)
	}
	tcpOpts := make([]coapTcpServer.Option, 0, 3)
	if serviceOpts.OnNewConnection != nil {
		tcpOpts = append(tcpOpts, options.WithOnNewConn(func(cc *coapTcpClient.Conn) {
			serviceOpts.OnNewConnection(cc)
		}))
	}
	if config.InactivityMonitor != nil {
		tcpOpts = append(tcpOpts, options.WithInactivityMonitor(config.InactivityMonitor.Timeout, func(cc *coapTcpClient.Conn) {
			serviceOpts.OnInactivityConnection(cc)
		}))
	}
	if config.KeepAlive != nil {
		tcpOpts = append(tcpOpts, options.WithKeepAlive(1, config.KeepAlive.Timeout, func(cc *coapTcpClient.Conn) {
			serviceOpts.OnInactivityConnection(cc)
		}))
	}
	for _, o := range opts {
		tcpOpts = append(tcpOpts, o)
	}
	return &tcpServer{
		coapServer: coapTcpServer.New(tcpOpts...),
		listener:   listener,
	}, nil
}
