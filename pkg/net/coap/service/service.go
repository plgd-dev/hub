package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	coapDtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/options"
	coapTcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
	coapUdpServer "github.com/plgd-dev/go-coap/v3/udp/server"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"go.opentelemetry.io/otel/trace"
)

func BlockWiseTransferSZXFromString(s string) (blockwise.SZX, error) {
	switch strings.ToLower(s) {
	case "16":
		return blockwise.SZX16, nil
	case "32":
		return blockwise.SZX32, nil
	case "64":
		return blockwise.SZX64, nil
	case "128":
		return blockwise.SZX128, nil
	case "256":
		return blockwise.SZX256, nil
	case "512":
		return blockwise.SZX512, nil
	case "1024":
		return blockwise.SZX1024, nil
	case "bert":
		return blockwise.SZXBERT, nil
	}
	return blockwise.SZX(0), fmt.Errorf("invalid value %v", s)
}

func closeOnError(services []service.APIService, logger log.Logger) {
	for _, service := range services {
		err := service.Close()
		if err != nil {
			logger.Errorf("cannot close service: %v", err)
		}
	}
}

func newService(protocol Protocol, config Config, serviceOpts Options, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...interface {
	coapTcpServer.Option
	coapDtlsServer.Option
	coapUdpServer.Option
},
) (service.APIService, error) {
	switch protocol {
	case TCP:
		coapServer, err := newTCPServer(config, serviceOpts, fileWatcher, logger, tracerProvider, opts...)
		if err != nil {
			return nil, fmt.Errorf("cannot create tcp server: %w", err)
		}
		return coapServer, nil
	case UDP:
		if config.TLS.IsEnabled() {
			coapServer, err := newDTLSServer(config, serviceOpts, fileWatcher, logger, tracerProvider, opts...)
			if err != nil {
				return nil, fmt.Errorf("cannot create dtls server: %w", err)
			}
			return coapServer, nil
		}
		coapServer, err := newUDPServer(config, serviceOpts, logger, opts...)
		if err != nil {
			return nil, fmt.Errorf("cannot create udp server: %w", err)
		}
		return coapServer, nil
	}
	return nil, nil
}

func makeOnInactivityConnection(logger log.Logger) func(conn mux.Conn) {
	return func(conn mux.Conn) {
		logger.Debugf("closing connection for inactivity: %v", conn.RemoteAddr())
		err := conn.Close()
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Errorf("cannot close connection %v for inactivity: %v", conn.RemoteAddr(), err)
		}
	}
}

// New creates server.
func New(ctx context.Context, config Config, router *mux.Router, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opt ...func(*Options)) (*service.Service, error) {
	err := config.Validate()
	if err != nil {
		return nil, err
	}
	serviceOpts := Options{
		MessagePool:            pool.New(config.MessagePoolSize, 1024),
		OnInactivityConnection: makeOnInactivityConnection(logger),
	}
	for _, o := range opt {
		o(&serviceOpts)
	}

	blockWiseTransferSZX := blockwise.SZX1024
	if config.BlockwiseTransfer.Enabled {
		blockWiseTransferSZX, err = BlockWiseTransferSZXFromString(config.BlockwiseTransfer.SZX)
		if err != nil {
			return nil, fmt.Errorf("blockWiseTransferSZX error: %w", err)
		}
	}

	opts := make([]interface {
		coapTcpServer.Option
		coapDtlsServer.Option
		coapUdpServer.Option
	}, 0, 8)
	opts = append(opts, options.WithBlockwise(config.BlockwiseTransfer.Enabled, blockWiseTransferSZX, config.GetTimeout()))
	opts = append(opts, options.WithMux(router))
	opts = append(opts, options.WithContext(ctx))
	opts = append(opts, options.WithMessagePool(serviceOpts.MessagePool))
	opts = append(opts, options.WithMaxMessageSize(config.MaxMessageSize))
	opts = append(opts, options.WithReceivedMessageQueueSize(config.MessageQueueSize))
	opts = append(opts, options.WithErrors(func(e error) {
		logger.Errorf("plgd/go-coap: %w", e)
	}))

	services := make([]service.APIService, 0, 2)
	for _, protocol := range config.Protocols {
		if protocol == UDP && !config.BlockwiseTransfer.Enabled {
			logger.Warnf("It's possible that UDP messages bigger than MTU (1500) will be dropped, since apis.coap.blockwiseTransfer.enabled is set to false.")
		}
		service, err := newService(protocol, config, serviceOpts, fileWatcher, logger, tracerProvider, opts...)
		if err != nil {
			closeOnError(services, logger)
			return nil, err
		}
		if service == nil {
			logger.Warnf("unsupported protocol(%v)", protocol)
			continue
		}
		services = append(services, service)
	}
	return service.New(services...), nil
}
