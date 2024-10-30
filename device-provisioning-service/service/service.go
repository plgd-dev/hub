package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/pion/dtls/v3"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/http"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/otelcoap"
	"github.com/plgd-dev/hub/v2/pkg/service"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	config                Config
	ctx                   context.Context
	cancel                context.CancelFunc
	messagePool           *pool.Pool
	linkedHubCache        *LinkedHubCache
	store                 *mongodb.Store
	logger                log.Logger
	authHandler           AuthHandler
	requestHandler        RequestHandler
	tracerProvider        trace.TracerProvider
	enrollmentGroupsCache *EnrollmentGroupsCache
}

const DPSTag = "dps"

func (server *Service) onInactivityConnection(cc mux.Conn) {
	session, ok := cc.Context().Value(clientKey).(*Session)
	errorf := server.logger.With(remoterAddr, cc.RemoteAddr().String()).Errorf
	warnf := server.logger.With(remoterAddr, cc.RemoteAddr().String()).Warnf
	if ok {
		errorf = session.Errorf
		warnf("cn: %v: keep alive was reached fail limit:: closing connection", session.String())
	} else {
		warnf("keep alive was reached fail limit:: closing connection")
	}
	if err := cc.Close(); err != nil && !errors.Is(err, context.Canceled) {
		errorf("failed to close connection: %w", err)
	}
}

type Options struct {
	authHandler    AuthHandler
	requestHandler RequestHandler
}

type Option func(o Options) Options

// Override default authorization handler
func WithAuthHandler(authHandler AuthHandler) Option {
	return func(o Options) Options {
		if authHandler != nil {
			o.authHandler = authHandler
		}
		return o
	}
}

// Override default request handler
func WithRequestHandler(requestHandler RequestHandler) Option {
	return func(o Options) Options {
		if requestHandler != nil {
			o.requestHandler = requestHandler
		}
		return o
	}
}

const serviceName = "device-provisioning-service"

func storePreConfiguredEnrollmentGroups(ctx context.Context, config Config, store *mongodb.Store) {
	create := func(g EnrollmentGroupConfig) {
		createCtx, cancel := context.WithTimeout(ctx, config.APIs.COAP.InactivityMonitor.Timeout)
		defer cancel()
		eg, hubs, err := g.ToProto()
		if err != nil {
			log.Warnf("cannot create pre configured enrollment group/hub: %v", err)
			return
		}
		err = store.UpsertEnrollmentGroup(createCtx, "", eg)
		if err != nil {
			log.Warnf("cannot store pre configured enrollment group: %v", err)
		}
		for _, hub := range hubs {
			err = store.UpsertHub(createCtx, "", hub)
			if err != nil {
				log.Warnf("cannot store pre configured hub: %v", err)
			}
		}
	}
	for _, g := range config.EnrollmentGroups {
		create(g)
	}
}

// New creates server.
func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, opts ...Option) (*service.Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector.Config, serviceName, fileWatcher, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	otelClient.AddCloseFunc(cancel)

	var closer fn.FuncList
	closer.AddFunc(otelClient.Close)
	tracerProvider := otelClient.GetTracerProvider()
	store, closeStore, err := NewStore(ctx, config.Clients.Storage.MongoDB, fileWatcher, logger, tracerProvider)
	if err != nil {
		closer.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	closer.AddFunc(closeStore)

	storePreConfiguredEnrollmentGroups(ctx, config, store)

	enrollmentGroupsCache := NewEnrollmentGroupsCache(ctx, config.Clients.Storage.CacheExpiration, store, logger)
	closer.AddFunc(enrollmentGroupsCache.Close)
	enrollmentGroupsRunner := periodic.New(ctx.Done(), config.Clients.Storage.CacheExpiration/2)
	enrollmentGroupsRunner(func(now time.Time) bool {
		enrollmentGroupsCache.CheckExpirations(now)
		return true
	})

	optCfg := Options{
		authHandler:    MakeDefaultAuthHandler(config, enrollmentGroupsCache),
		requestHandler: RequestHandle{},
	}
	for _, o := range opts {
		optCfg = o(optCfg)
	}
	runner := periodic.New(ctx.Done(), config.APIs.COAP.InactivityMonitor.Timeout/2)
	runner(func(now time.Time) bool {
		optCfg.authHandler.GetChainsCache().CheckExpirations(now)
		return true
	})

	var httpService *http.Service
	if config.APIs.HTTP.Enabled {
		httpService, err = http.New(ctx, serviceName, config.APIs.HTTP.Config, fileWatcher, logger, tracerProvider, store)
		if err != nil {
			closer.Execute()
			return nil, fmt.Errorf("cannot create http service: %w", err)
		}
	}

	linkedHubCache := NewLinkedHubCache(ctx, config.Clients.Storage.CacheExpiration, store, fileWatcher, logger, tracerProvider)
	s := Service{
		config:         config,
		linkedHubCache: linkedHubCache,

		ctx:    ctx,
		cancel: cancel,

		messagePool:           pool.New(config.APIs.COAP.MessagePoolSize, 1024),
		store:                 store,
		logger:                logger,
		authHandler:           optCfg.authHandler,
		requestHandler:        optCfg.requestHandler,
		tracerProvider:        tracerProvider,
		enrollmentGroupsCache: enrollmentGroupsCache,
	}

	ss, err := s.createServices(fileWatcher, logger, tracerProvider)
	if err != nil {
		if httpService != nil {
			httpService.Close()
		}
		closer.Execute()
		return nil, fmt.Errorf("cannot create coap services: %w", err)
	}
	if httpService != nil {
		ss.Add(httpService)
	}
	ss.AddCloseFunc(closer.Execute)
	return ss, nil
}

func (RequestHandle) DefaultHandler(_ context.Context, req *mux.Message, _ *Session, _ []*LinkedHub, _ *EnrollmentGroup) (*pool.Message, error) {
	path, _ := req.Options().Path()

	/*
		switch {
		case strings.HasPrefix("/"+path, uri.ResourceRoute):
			resourceRouteHandler(req, session)
		default:
			return nil, statusErrorf(coapCodes.NotFound, "unknown path %v", path)
		}
	*/
	return nil, statusErrorf(coapCodes.NotFound, "unknown path %v", path)
}

const clientKey = "client"

func (server *Service) getVerifiedChain(ctx context.Context, conn net.Conn) (verifiedChains [][]*x509.Certificate) {
	var certRaw []byte
	switch tlsCon := conn.(type) {
	case *tls.Conn:
		if len(tlsCon.ConnectionState().PeerCertificates) > 0 {
			certRaw = tlsCon.ConnectionState().PeerCertificates[0].Raw
		}
	case *dtls.Conn:
		if err := tlsCon.HandshakeContext(ctx); err != nil {
			server.logger.With(remoterAddr, conn.RemoteAddr().String()).Errorf("cannot get connection state: handshake failed: %w", err)
			return nil
		}

		cs, ok := tlsCon.ConnectionState()
		if !ok {
			server.logger.With(remoterAddr, conn.RemoteAddr().String()).Errorf("cannot get connection state")
			return nil
		}
		if len(cs.PeerCertificates) > 0 {
			certRaw = cs.PeerCertificates[0]
		}
	default:
		server.logger.With(remoterAddr, conn.RemoteAddr().String()).Errorf("unknown connection type: %T", conn)
		return nil
	}
	if len(certRaw) == 0 {
		server.logger.With(remoterAddr, conn.RemoteAddr().String()).Debugf("cannot get verified chain: peer certificates are empty")
		return nil
	}
	id := toCRC64(certRaw)
	v := server.authHandler.GetChainsCache().Load(id)
	if v != nil {
		return v.Data()
	}
	server.logger.With(remoterAddr, conn.RemoteAddr().String()).Debugf("cannot get verified chain: it is not set")
	return nil
}

func (server *Service) coapConnOnNew(coapConn mux.Conn) {
	verifiedChains := server.getVerifiedChain(server.ctx, coapConn.NetConn())
	session := newSession(server, coapConn, verifiedChains)
	coapConn.SetContextValue(clientKey, session)
	coapConn.AddOnClose(func() {
		session.OnClose()
	})
}

func (server *Service) toInternalHandler(w mux.ResponseWriter, r *mux.Message, h func(ctx context.Context, req *mux.Message, session *Session) (*pool.Message, error)) {
	session, ok := w.Conn().Context().Value(clientKey).(*Session)
	if !ok {
		addr := w.Conn().RemoteAddr().String()
		log.Errorf("unknown session %v", addr)
		return
	}
	startTime := time.Now()
	path, _ := r.Options().Path()
	ctx, span := otelcoap.Start(r.Context(), path, r.Code().String(), otelcoap.WithTracerProvider(server.tracerProvider), otelcoap.WithSpanOptions(trace.WithSpanKind(trace.SpanKindServer)))
	defer span.End()
	r.SetContext(ctx)
	otelcoap.MessageReceivedEvent(ctx, otelcoap.MakeMessage(r.Message))

	ctx, cancel := context.WithTimeout(r.Context(), session.server.config.APIs.COAP.InactivityMonitor.Timeout)
	defer cancel()
	resp, errResp := h(ctx, r, session)
	if errResp != nil {
		s, ok := status.FromError(errResp)
		if ok {
			m, cleanUp := coapgwMessage.GetErrorResponse(ctx, server.messagePool, s.Code(), r.Token(), errResp)
			defer cleanUp()
			resp = m
		}
		defer func() {
			span.RecordError(errResp)
			span.SetStatus(otelCodes.Error, errResp.Error())
			_ = session.Close()
		}()
	}
	if resp != nil {
		otelMsg := otelcoap.MakeMessage(resp)
		if err := session.WriteMessage(resp); err != nil {
			session.Errorf("cannot send error: %w", err)
		}
		otelcoap.MessageSentEvent(r.Context(), otelMsg)
		span.SetAttributes(otelcoap.StatusCodeAttr(resp.Code()))
	}
	session.logRequestResponse(ctx, startTime, r, resp, errResp)
}

func statusErrorf(code coapCodes.Code, fmt string, args ...interface{}) error {
	return status.Errorf(NewMessageWithCode(code), fmt, args...)
}

func (server *Service) toHandler(h func(ctx context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		server.toInternalHandler(w, r, func(ctx context.Context, _ *mux.Message, session *Session) (*pool.Message, error) {
			session.resolveLocalEndpoints()
			if err := session.checkForError(); err != nil {
				p, _ := r.Options().Path()
				return nil, statusErrorf(coapCodes.Forbidden, "cannot process %v %v: %w", r.Code(), p, err)
			}
			linkedHubs, group, err := session.getGroupAndLinkedHubs(ctx)
			if err != nil {
				p, _ := r.Options().Path()
				return nil, statusErrorf(coapCodes.BadRequest, "cannot process %v %v: %w", r.Code(), p, err)
			}
			return h(ctx, r, session, linkedHubs, group)
		})
	})
}

// createServices setups coap server
func (server *Service) createServices(fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*service.Service, error) {
	setHandlerError := func(uri string, err error) error {
		return fmt.Errorf("failed to set %v handler: %w", uri, err)
	}

	m := mux.NewRouter()
	m.DefaultHandle(server.toHandler(server.requestHandler.DefaultHandler))

	if err := m.Handle(plgdtime.ResourceURI, server.toHandler(server.requestHandler.ProcessPlgdTime)); err != nil {
		return nil, setHandlerError(plgdtime.ResourceURI, err)
	}
	if err := m.Handle(uri.Ownership, server.toHandler(server.requestHandler.ProcessOwnership)); err != nil {
		return nil, setHandlerError(uri.Ownership, err)
	}
	if err := m.Handle(uri.Credentials, server.toHandler(server.requestHandler.ProcessCredentials)); err != nil {
		return nil, setHandlerError(uri.Credentials, err)
	}
	if err := m.Handle(uri.ACLs, server.toHandler(server.requestHandler.ProcessACLs)); err != nil {
		return nil, setHandlerError(uri.ACLs, err)
	}
	if err := m.Handle(uri.CloudConfiguration, server.toHandler(server.requestHandler.ProcessCloudConfiguration)); err != nil {
		return nil, setHandlerError(uri.CloudConfiguration, err)
	}

	return coapService.New(server.ctx, server.config.APIs.COAP.Config, m, fileWatcher, logger, tracerProvider,
		coapService.WithOnNewConnection(server.coapConnOnNew),
		coapService.WithOnInactivityConnection(server.onInactivityConnection),
		coapService.WithMessagePool(server.messagePool),
		coapService.WithOverrideTLS(func(cfg *tls.Config, _ coapService.VerifyByCRL) *tls.Config {
			cfg.InsecureSkipVerify = true
			cfg.ClientAuth = tls.RequireAnyClientCert
			cfg.VerifyPeerCertificate = server.authHandler.VerifyPeerCertificate
			cfg.VerifyConnection = server.authHandler.VerifyConnection
			return cfg
		}),
	)
}
