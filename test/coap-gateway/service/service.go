package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	coapgwUri "github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
)

// Service is a configuration of coap-gateway
type Service struct {
	config      Config
	coapServer  *tcp.Server
	listener    tcp.Listener
	closeFn     func()
	ctx         context.Context
	cancel      context.CancelFunc
	sigs        chan os.Signal
	taskQueue   *queue.Queue
	makeHandler MakeServiceHandler
	clients     []*Client
}

func newTCPListener(config COAPConfig, fileWatcher *fsnotify.Watcher, logger log.Logger) (tcp.Listener, func(), error) {
	if !config.TLS.Enabled {
		listener, err := net.NewTCPListener("tcp", config.Addr)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot create tcp listener: %w", err)
		}
		closeListener := func() {
			if err := listener.Close(); err != nil {
				log.Errorf("failed to close tcp listener: %w", err)
			}
		}
		return listener, closeListener, nil
	}

	var closeListener fn.FuncList
	coapsTLS, err := certManagerServer.New(config.TLS.Config, fileWatcher, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create tls cert manager: %w", err)
	}
	closeListener.AddFunc(coapsTLS.Close)
	listener, err := net.NewTLSListener("tcp", config.Addr, coapsTLS.GetTLSConfig())
	if err != nil {
		closeListener.Execute()
		return nil, nil, fmt.Errorf("cannot create tcp-tls listener: %w", err)
	}
	closeListener.AddFunc(func() {
		if err := listener.Close(); err != nil {
			log.Errorf("failed to close tcp-tls listener: %w", err)
		}
	})
	return listener, closeListener.ToFunction(), nil
}

// New creates server.
func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, makeHandler MakeServiceHandler) (*Service, error) {
	queue, err := queue.New(config.TaskQueue)
	if err != nil {
		return nil, fmt.Errorf("cannot create job queue %w", err)
	}

	var closeFn fn.FuncList
	listener, closeListener, err := newTCPListener(config.APIs.COAP, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create listener: %w", err)
	}
	closeFn.AddFunc(closeListener)

	ctx, cancel := context.WithCancel(ctx)

	s := Service{
		config:      config,
		listener:    listener,
		closeFn:     closeFn.ToFunction(),
		ctx:         ctx,
		cancel:      cancel,
		sigs:        make(chan os.Signal, 1),
		taskQueue:   queue,
		makeHandler: makeHandler,
	}

	if err := s.setupCoapServer(); err != nil {
		return nil, fmt.Errorf("cannot setup coap server: %w", err)
	}

	return &s, nil
}

func decodeMsgToDebug(client *Client, resp *pool.Message, tag string) {
	if !client.server.config.Log.DumpCoapMessages {
		return
	}
	msg := message.ToJson(resp, true, true)
	log.Get().With("msg", msg).Debug(tag)
}

const clientKey = "client"

func (s *Service) coapConnOnNew(coapConn *tcp.ClientConn, tlscon *tls.Conn) {
	client := newClient(s, coapConn, s.makeHandler(s, WithCoapConnectionOpt(coapConn)))
	coapConn.SetContextValue(clientKey, client)
	coapConn.AddOnClose(func() {
		client.OnClose()
	})
	s.clients = append(s.clients, client)
}

func (s *Service) loggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client, ok := w.Client().Context().Value(clientKey).(*Client)
		if !ok {
			client = newClient(s, w.Client().ClientConn().(*tcp.ClientConn), nil)
		}
		tmp, err := pool.New(0, 0).ConvertFrom(r.Message)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot convert from mux.Message: %w", err), coapCodes.InternalServerError, r.Token)
			return
		}
		decodeMsgToDebug(client, tmp, "RECEIVED-COMMAND")
		next.ServeCOAP(w, r)
	})
}

func validateCommand(s mux.ResponseWriter, req *mux.Message, server *Service, fnc func(req *mux.Message, client *Client)) {
	client, ok := s.Client().Context().Value(clientKey).(*Client)
	if !ok || client == nil {
		client = newClient(server, s.Client().ClientConn().(*tcp.ClientConn), nil)
	}
	closeClient := func(c *Client) {
		if err := c.Close(); err != nil {
			log.Errorf("cannot handle command: %w", err)
		}
	}
	err := server.Submit(func() {
		switch req.Code {
		case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
			fnc(req, client)
		case coapCodes.Empty:
			if !ok {
				client.logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), coapCodes.InternalServerError, req.Token)
				closeClient(client)
				return
			}
		case coapCodes.Content:
			// Unregistered observer at a peer send us a notification
			tmp, err := pool.New(0, 0).ConvertFrom(req.Message)
			if err != nil {
				log.Errorf("cannot convert dropped notification: %w", err)
				return
			}
			decodeMsgToDebug(client, tmp, "DROPPED-NOTIFICATION")
		default:
			log.Errorf("received invalid code: CoapCode(%v)", req.Code)
		}
	})
	if err != nil {
		closeClient(client)
		log.Errorf("cannot handle request %v by task queue: %w", req.String(), err)
	}
}

func defaultHandler(req *mux.Message, client *Client) {
	path, _ := req.Options.Path()
	client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: unknown path %v", client.GetDeviceID(), path), coapCodes.NotFound, req.Token)
}

func (s *Service) setupCoapServer() error {
	setHandlerError := func(uri string, err error) error {
		return fmt.Errorf("failed to set %v handler: %w", uri, err)
	}
	m := mux.NewRouter()
	m.Use(s.loggingMiddleware)
	m.DefaultHandle(mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, defaultHandler)
	}))
	if err := m.Handle(coapgwUri.ResourceDirectory, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, resourceDirectoryHandler)
	})); err != nil {
		return setHandlerError(coapgwUri.ResourceDirectory, err)
	}
	if err := m.Handle(coapgwUri.SignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, signUpHandler)
	})); err != nil {
		return setHandlerError(coapgwUri.SignUp, err)
	}
	if err := m.Handle(coapgwUri.SignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, signInHandler)
	})); err != nil {
		return setHandlerError(coapgwUri.SignIn, err)
	}
	if err := m.Handle(coapgwUri.RefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, refreshTokenHandler)
	})); err != nil {
		return setHandlerError(coapgwUri.RefreshToken, err)
	}

	opts := make([]tcp.ServerOption, 0, 6)
	opts = append(opts, tcp.WithOnNewClientConn(s.coapConnOnNew))
	opts = append(opts, tcp.WithMux(m))
	opts = append(opts, tcp.WithContext(s.ctx))
	opts = append(opts, tcp.WithErrors(func(e error) {
		log.Errorf("plgd/test-coap: %w", e)
	}))
	opts = append(opts, tcp.WithGoPool(func(f func()) error {
		// we call directly function in connection-goroutine because
		// pairing request/response cannot be done in taskQueue for a observe resource.
		// - the observe resource creates task which wait for the response and this wait can be infinite
		// if all task goroutines are processing observations and they are waiting for the responses, which
		// will be stored in task queue.  it happens when we use task queue here.
		f()
		return nil
	}))
	s.coapServer = tcp.NewServer(opts...)
	return nil
}

func (s *Service) Serve() error {
	return s.serveWithHandlingSignal()
}

func (s *Service) serveWithHandlingSignal() error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(server *Service) {
		defer wg.Done()
		err = server.coapServer.Serve(server.listener)
		server.cancel()
		server.closeFn()
	}(s)

	signal.Notify(s.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-s.sigs

	s.coapServer.Stop()
	wg.Wait()

	return err
}

func (s *Service) Submit(task func()) error {
	return s.taskQueue.Submit(task)
}

func (s *Service) GetClients() []*Client {
	return s.clients
}

// Close turns off the server.
func (s *Service) Close() error {
	select {
	case s.sigs <- syscall.SIGTERM:
	default:
	}
	return nil
}
