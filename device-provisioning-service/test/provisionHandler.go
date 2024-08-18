package test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.uber.org/atomic"
	"go.uber.org/zap/zapcore"
)

type RequestHandlerWithDps struct {
	t           *testing.T
	lock        sync.Mutex
	dpsCfg      service.Config
	dpsShutdown func()
	logger      log.Logger
	started     bool
}

func (h *RequestHandlerWithDps) T() *testing.T {
	return h.t
}

func (h *RequestHandlerWithDps) Cfg() service.Config {
	return h.dpsCfg
}

func (h *RequestHandlerWithDps) IsStarted() bool {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.started
}

func (h *RequestHandlerWithDps) StartDps(opts ...service.Option) {
	if h.dpsShutdown != nil {
		panic("dps already started")
	}
	h.Logf("start provisioning")
	h.dpsShutdown = New(h.t, h.dpsCfg, opts...)
	h.lock.Lock()
	h.started = true
	h.lock.Unlock()
}

func (h *RequestHandlerWithDps) StopDps() {
	h.Logf("stop provisioning")
	var dpsShutdown func()
	h.lock.Lock()
	dpsShutdown = h.dpsShutdown
	h.dpsShutdown = nil
	h.lock.Unlock()
	if dpsShutdown != nil {
		dpsShutdown()
	}
}

func (h *RequestHandlerWithDps) RestartDps(opts ...service.Option) {
	h.StopDps()
	h.Logf("restart provisioning")
	dpsShutdown := New(h.t, h.dpsCfg, opts...)
	h.lock.Lock()
	h.dpsShutdown = dpsShutdown
	h.lock.Unlock()
}

func (h *RequestHandlerWithDps) Logf(template string, args ...interface{}) {
	h.logger.Debugf(template, args...)
}

func MakeRequestHandlerWithDps(t *testing.T, dpsCfg service.Config) RequestHandlerWithDps {
	return RequestHandlerWithDps{
		t:      t,
		dpsCfg: dpsCfg,
		logger: log.NewLogger(log.Config{Level: zapcore.DebugLevel}),
	}
}

type HandlerID int

const (
	HandlerIDDefault HandlerID = iota
	HandlerIDTime
	HandlerIDOwnership
	HandlerIDCredentials
	HandlerIDACLs
	HandlerIDCloudConfiguration
)

type (
	CheckCountFn       func(h HandlerID, count uint64) codes.Code
	CheckFinalCountsFn func(defaultHandlerCount, processTimeCount, processOwnershipCount, processCredentialsCount,
		processACLsCount, processCloudConfigurationCount uint64) (bool, error)
)

type RequestHandlerWithCounter struct {
	RequestHandlerWithDps
	// request callbacks
	defaultHandlerCounter            atomic.Uint64
	processOwnershipCounter          atomic.Uint64
	processTimeCounter               atomic.Uint64
	processACLsCounter               atomic.Uint64
	processCloudConfigurationCounter atomic.Uint64
	processCredentialsCounter        atomic.Uint64
	checkCount                       CheckCountFn       // get return code for given handler based on the number of previous failures
	checkFinalCounts                 CheckFinalCountsFn // verify final handler failure counter values
	r                                service.RequestHandle
}

func NewRequestHandlerWithCounter(t *testing.T, dpsCfg service.Config, checkCount CheckCountFn, checkFinalCounts CheckFinalCountsFn) *RequestHandlerWithCounter {
	return &RequestHandlerWithCounter{
		RequestHandlerWithDps: MakeRequestHandlerWithDps(t, dpsCfg),
		checkCount:            checkCount,
		checkFinalCounts:      checkFinalCounts,
	}
}

// we know the counter is invalid when the value is higher
func NewRequestHandlerWithExpectedCounters(t *testing.T, failLimit uint64, failCode codes.Code, expectedTimeCount, expectedOwnershipCount,
	expectedCloudConfigurationCount, expectedCredentialsCount, expectedACLsCount uint64,
) *RequestHandlerWithCounter {
	dpsCfg := MakeConfig(t)
	return NewRequestHandlerWithCounter(t, dpsCfg, func(h HandlerID, count uint64) codes.Code {
		switch h {
		case HandlerIDTime,
			HandlerIDOwnership,
			HandlerIDCloudConfiguration,
			HandlerIDCredentials,
			HandlerIDACLs:
			if count < failLimit {
				return failCode
			}
		case HandlerIDDefault:
		}
		return 0
	}, func(defaultHandlerCount, processTimeCount, processOwnershipCount, processCloudConfigurationCount, processCredentialsCount, processACLsCount uint64) (bool, error) {
		if defaultHandlerCount > 0 ||
			(processTimeCount > expectedTimeCount) ||
			(processOwnershipCount > expectedOwnershipCount) ||
			(processCloudConfigurationCount > expectedCloudConfigurationCount) ||
			(processCredentialsCount > expectedCredentialsCount) ||
			(processACLsCount > expectedACLsCount) {
			return false,
				fmt.Errorf("invalid counters: defaultHandlerCounter=%v expectedTimeCount=%v(exp %v)	processOwnershipCounter=%v(exp %v) processCloudConfigurationCounter=%v(exp %v) processCredentialsCounter=%v(exp %v) processACLsCounter=%v(exp %v) ",
					defaultHandlerCount,
					processTimeCount, expectedTimeCount,
					processOwnershipCount, expectedOwnershipCount,
					processCloudConfigurationCount, expectedCloudConfigurationCount,
					processCredentialsCount, expectedCredentialsCount,
					processACLsCount, expectedACLsCount,
				)
		}
		return defaultHandlerCount == 0 &&
			(processTimeCount == expectedTimeCount) &&
			(processOwnershipCount == expectedOwnershipCount) &&
			(processCloudConfigurationCount == expectedCloudConfigurationCount) &&
			(processCredentialsCount == expectedCredentialsCount) &&
			(processACLsCount == expectedACLsCount), nil
	})
}

func (h *RequestHandlerWithCounter) DefaultHandler(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if !h.IsStarted() {
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "DefaultHandler: DPS not started")
	}

	c := h.defaultHandlerCounter.Load()
	if h.checkCount != nil {
		if code := h.checkCount(HandlerIDDefault, c); code != 0 {
			h.defaultHandlerCounter.Inc()
			return nil, status.Errorf(service.NewMessageWithCode(code), "DefaultHandler: force retry")
		}
	}
	msg, err := h.r.DefaultHandler(ctx, req, session, linkedHubs, group)
	if err != nil {
		h.Logf("DefaultHandler: %s", err.Error())
		// internal error -> return ServiceUnavailable to force a single step retry
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "DefaultHandler: %s", err.Error())
	}
	h.defaultHandlerCounter.Inc()
	h.Logf("DefaultHandler: %v", h.defaultHandlerCounter.Load())
	return msg, nil
}

func (h *RequestHandlerWithCounter) ProcessPlgdTime(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if !h.IsStarted() {
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessPlgdTime DPS not started")
	}

	c := h.processTimeCounter.Load()
	if h.checkCount != nil {
		if code := h.checkCount(HandlerIDTime, c); code != 0 {
			h.processTimeCounter.Inc()
			h.Logf("ProcessPlgdTime: %v", h.processTimeCounter.Load())
			return nil, status.Errorf(service.NewMessageWithCode(code), "ProcessPlgdTime: force retry")
		}
	}
	msg, err := h.r.ProcessPlgdTime(ctx, req, session, linkedHubs, group)
	if err != nil {
		h.Logf("ProcessPlgdTime: %s", err.Error())
		// internal error -> return ServiceUnavailable to force a single step retry
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessPlgdTime: %s", err.Error())
	}
	h.processTimeCounter.Inc()
	h.Logf("ProcessPlgdTime: %v", h.processTimeCounter.Load())
	return msg, nil
}

func (h *RequestHandlerWithCounter) ProcessOwnership(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if !h.IsStarted() {
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessOwnership: DPS not started")
	}

	c := h.processOwnershipCounter.Load()
	if h.checkCount != nil {
		if code := h.checkCount(HandlerIDOwnership, c); code != 0 {
			h.processOwnershipCounter.Inc()
			h.Logf("ProcessOwnership: %v", h.processOwnershipCounter.Load())
			return nil, status.Errorf(service.NewMessageWithCode(code), "ProcessOwnership: force retry")
		}
	}
	msg, err := h.r.ProcessOwnership(ctx, req, session, linkedHubs, group)
	if err != nil {
		h.Logf("ProcessOwnership: %s", err.Error())
		// internal error -> return ServiceUnavailable to force a single step retry
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessOwnership: %s", err.Error())
	}
	h.processOwnershipCounter.Inc()
	h.Logf("ProcessOwnership: %v", h.processOwnershipCounter.Load())
	return msg, nil
}

func (h *RequestHandlerWithCounter) ProcessCredentials(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if !h.IsStarted() {
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessCredentials: DPS not started")
	}

	c := h.processCredentialsCounter.Load()
	if h.checkCount != nil {
		if code := h.checkCount(HandlerIDCredentials, c); code != 0 {
			h.processCredentialsCounter.Inc()
			h.Logf("ProcessCredentials: %v", h.processCredentialsCounter.Load())
			return nil, status.Errorf(service.NewMessageWithCode(code), "ProcessCredentials: force retry")
		}
	}
	msg, err := h.r.ProcessCredentials(ctx, req, session, linkedHubs, group)
	if err != nil {
		h.Logf("ProcessCredentials: %s", err.Error())
		// internal error -> return ServiceUnavailable to force a single step retry
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessCredentials: %s", err.Error())
	}
	h.processCredentialsCounter.Inc()
	h.Logf("ProcessCredentials: %v", h.processCredentialsCounter.Load())
	return msg, nil
}

func (h *RequestHandlerWithCounter) ProcessACLs(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if !h.IsStarted() {
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessACLs: DPS not started")
	}

	c := h.processACLsCounter.Load()
	if h.checkCount != nil {
		if code := h.checkCount(HandlerIDACLs, c); code != 0 {
			h.processACLsCounter.Inc()
			h.Logf("ProcessACLs: %v", h.processACLsCounter.Load())
			return nil, status.Errorf(service.NewMessageWithCode(code), "ProcessACLs: force retry")
		}
	}
	msg, err := h.r.ProcessACLs(ctx, req, session, linkedHubs, group)
	if err != nil {
		h.Logf("ProcessACLs: %s", err.Error())
		// internal error -> return ServiceUnavailable to force a single step retry
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessACLs: %s", err.Error())
	}
	h.processACLsCounter.Inc()
	h.Logf("ProcessACLs: %v", h.processACLsCounter.Load())
	return msg, nil
}

func (h *RequestHandlerWithCounter) ProcessCloudConfiguration(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if !h.IsStarted() {
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessCloudConfiguration: DPS not started")
	}

	c := h.processCloudConfigurationCounter.Load()
	if h.checkCount != nil {
		if code := h.checkCount(HandlerIDCloudConfiguration, c); code != 0 {
			h.processCloudConfigurationCounter.Inc()
			h.Logf("ProcessCloudConfiguration: %v", h.processCloudConfigurationCounter.Load())
			return nil, status.Errorf(service.NewMessageWithCode(code), "ProcessCloudConfiguration: force retry")
		}
	}
	msg, err := h.r.ProcessCloudConfiguration(ctx, req, session, linkedHubs, group)
	if err != nil {
		h.Logf("ProcessCloudConfiguration: %w", err)
		// internal error -> return ServiceUnavailable to force a single step retry
		return nil, status.Errorf(service.NewMessageWithCode(codes.ServiceUnavailable), "ProcessCloudConfiguration: %s", err.Error())
	}
	h.processCloudConfigurationCounter.Inc()
	h.Logf("ProcessCloudConfiguration: %v", h.processCloudConfigurationCounter.Load())
	return msg, nil
}

func (h *RequestHandlerWithCounter) encode() string {
	return "defaultHandlerCounter=" + strconv.FormatUint(h.defaultHandlerCounter.Load(), 10) + " " +
		"processTimeCounter=" + strconv.FormatUint(h.processTimeCounter.Load(), 10) + " " +
		"processOwnershipCounter=" + strconv.FormatUint(h.processOwnershipCounter.Load(), 10) + " " +
		"processCloudConfigurationCounter=" + strconv.FormatUint(h.processCloudConfigurationCounter.Load(), 10) + " " +
		"processCredentialsCounter=" + strconv.FormatUint(h.processCredentialsCounter.Load(), 10) + " " +
		"processACLsCounter=" + strconv.FormatUint(h.processACLsCounter.Load(), 10)
}

func (h *RequestHandlerWithCounter) Verify(ctx context.Context) error {
	logCounter := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("unexpected counters %s", h.encode())
		case <-time.After(time.Second):
			logCounter++
			if logCounter%3 == 0 {
				h.Logf(h.encode())
			}
		}
		done, err := h.checkFinalCounts(h.defaultHandlerCounter.Load(),
			h.processTimeCounter.Load(),
			h.processOwnershipCounter.Load(),
			h.processCloudConfigurationCounter.Load(),
			h.processCredentialsCounter.Load(),
			h.processACLsCounter.Load())
		if err != nil {
			return fmt.Errorf("counter verification failed: %w", err)
		}
		if done {
			return nil
		}
	}
}
