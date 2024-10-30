package service

import (
	"crypto/tls"
	"fmt"

	"github.com/pion/dtls/v3"
	coapDtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	coapTcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
	coapUdpClient "github.com/plgd-dev/go-coap/v3/udp/client"
	coapUdpServer "github.com/plgd-dev/go-coap/v3/udp/server"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"go.opentelemetry.io/otel/trace"
)

type dtlsListerner struct {
	coapDtlsServer.Listener
	close func()
}

type dtlsServer struct {
	coapServer *coapDtlsServer.Server
	listener   *dtlsListerner
}

func (s *dtlsServer) Serve() error {
	return s.coapServer.Serve(s.listener)
}

func (s *dtlsServer) Close() error {
	s.coapServer.Stop()
	s.listener.close()
	return nil
}

type udpListerner struct {
	*net.UDPConn
	close func()
}

type udpServer struct {
	coapServer *coapUdpServer.Server
	listener   *udpListerner
}

func (s *udpServer) Serve() error {
	return s.coapServer.Serve(s.listener.UDPConn)
}

func (s *udpServer) Close() error {
	s.coapServer.Stop()
	s.listener.close()
	return nil
}

func newUDPListener(config Config, logger log.Logger) (*udpListerner, error) {
	listener, err := net.NewListenUDP("udp", config.Addr)
	if err != nil {
		return nil, fmt.Errorf("cannot create tcp listener: %w", err)
	}
	closeListener := func() {
		if err := listener.Close(); err != nil {
			logger.Errorf("failed to close tcp listener: %w", err)
		}
	}
	return &udpListerner{
		UDPConn: listener,
		close:   closeListener,
	}, nil
}

var mapDTLSClientAuth = map[tls.ClientAuthType]dtls.ClientAuthType{
	tls.NoClientCert:               dtls.NoClientCert,
	tls.RequestClientCert:          dtls.RequestClientCert,
	tls.RequireAnyClientCert:       dtls.RequireAnyClientCert,
	tls.VerifyClientCertIfGiven:    dtls.VerifyClientCertIfGiven,
	tls.RequireAndVerifyClientCert: dtls.RequireAndVerifyClientCert,
}

func toDTLSClientAuth(t tls.ClientAuthType) dtls.ClientAuthType {
	return mapDTLSClientAuth[t]
}

func TLSConfigToDTLSConfig(tlsConfig *tls.Config) *dtls.Config {
	var getClientCertificate func(cri *dtls.CertificateRequestInfo) (*tls.Certificate, error)
	if tlsConfig.GetClientCertificate != nil {
		getClientCertificate = func(cri *dtls.CertificateRequestInfo) (*tls.Certificate, error) {
			return tlsConfig.GetClientCertificate(&tls.CertificateRequestInfo{AcceptableCAs: cri.AcceptableCAs})
		}
	}
	var getCertificate func(chi *dtls.ClientHelloInfo) (*tls.Certificate, error)
	if tlsConfig.GetCertificate != nil {
		getCertificate = func(chi *dtls.ClientHelloInfo) (*tls.Certificate, error) {
			return tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: chi.ServerName})
		}
	}
	return &dtls.Config{
		GetCertificate:        getCertificate,
		ClientCAs:             tlsConfig.ClientCAs,
		VerifyPeerCertificate: tlsConfig.VerifyPeerCertificate,
		RootCAs:               tlsConfig.RootCAs,
		InsecureSkipVerify:    tlsConfig.InsecureSkipVerify,
		Certificates:          tlsConfig.Certificates,
		ServerName:            tlsConfig.ServerName,
		GetClientCertificate:  getClientCertificate,
		ClientAuth:            toDTLSClientAuth(tlsConfig.ClientAuth),
		CipherSuites:          []dtls.CipherSuiteID{dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM, dtls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, dtls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
	}
}

func newDTLSListener(config Config, serviceOpts Options, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*dtlsListerner, error) {
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
	dtlsCfg := TLSConfigToDTLSConfig(tlsCfg)
	dtlsCfg.LoggerFactory = logger.DTLSLoggerFactory()
	listener, err := net.NewDTLSListener("udp", config.Addr, dtlsCfg)
	if err != nil {
		closeListener.Execute()
		return nil, fmt.Errorf("cannot create dtls listener: %w", err)
	}
	closeListener.AddFunc(func() {
		if err := listener.Close(); err != nil {
			logger.Errorf("failed to close dtls listener: %w", err)
		}
	})

	return &dtlsListerner{
		Listener: listener,
		close:    closeListener.ToFunction(),
	}, nil
}

func newDTLSServer(config Config, serviceOpts Options, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...interface {
	coapTcpServer.Option
	coapDtlsServer.Option
	coapUdpServer.Option
},
) (*dtlsServer, error) {
	listener, err := newDTLSListener(config, serviceOpts, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create listener: %w", err)
	}
	dtlsOpts := make([]coapDtlsServer.Option, 0, 4)
	if serviceOpts.OnNewConnection != nil {
		dtlsOpts = append(dtlsOpts, options.WithOnNewConn(func(coapConn *coapUdpClient.Conn) {
			serviceOpts.OnNewConnection(coapConn)
		}))
	}
	if config.InactivityMonitor != nil {
		dtlsOpts = append(dtlsOpts, options.WithInactivityMonitor(config.InactivityMonitor.Timeout, func(cc *coapUdpClient.Conn) {
			serviceOpts.OnInactivityConnection(cc)
		}), options.WithTransmission(1, config.InactivityMonitor.Timeout, 2))
	}
	if config.KeepAlive != nil {
		dtlsOpts = append(dtlsOpts, options.WithKeepAlive(1, config.KeepAlive.Timeout, func(cc *coapUdpClient.Conn) {
			serviceOpts.OnInactivityConnection(cc)
		}), options.WithTransmission(1, config.KeepAlive.Timeout, 2))
	}
	for _, o := range opts {
		dtlsOpts = append(dtlsOpts, o)
	}
	return &dtlsServer{
		coapServer: coapDtlsServer.New(dtlsOpts...),
		listener:   listener,
	}, nil
}

func newUDPServer(config Config, serviceOpts Options, logger log.Logger, opts ...interface {
	coapTcpServer.Option
	coapDtlsServer.Option
	coapUdpServer.Option
},
) (*udpServer, error) {
	listener, err := newUDPListener(config, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create listener: %w", err)
	}
	udpOpts := make([]coapUdpServer.Option, 0, 4)
	if serviceOpts.OnNewConnection != nil {
		udpOpts = append(udpOpts, options.WithOnNewConn(func(coapConn *coapUdpClient.Conn) {
			serviceOpts.OnNewConnection(coapConn)
		}))
	}
	if config.InactivityMonitor != nil {
		udpOpts = append(udpOpts, options.WithInactivityMonitor(config.InactivityMonitor.Timeout, func(cc *coapUdpClient.Conn) {
			serviceOpts.OnNewConnection(cc)
		}), options.WithTransmission(1, config.InactivityMonitor.Timeout, 2))
	}
	if config.KeepAlive != nil {
		udpOpts = append(udpOpts, options.WithKeepAlive(2, config.KeepAlive.Timeout, func(cc *coapUdpClient.Conn) {
			serviceOpts.OnInactivityConnection(cc)
		}), options.WithTransmission(1, config.KeepAlive.Timeout, 2))
	}
	for _, o := range opts {
		udpOpts = append(udpOpts, o)
	}
	return &udpServer{
		coapServer: coapUdpServer.New(udpOpts...),
		listener:   listener,
	}, nil
}
