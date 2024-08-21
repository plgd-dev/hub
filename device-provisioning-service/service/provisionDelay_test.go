package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"go.uber.org/atomic"
)

type testRequestHandlerWithDelay struct {
	test.RequestHandlerWithDps
	delayedTime               atomic.Bool
	waitForTime               chan struct{}
	delayedOwnership          atomic.Bool
	waitForOwnership          chan struct{}
	delayedCloudConfiguration atomic.Bool
	waitForCloudConfiguration chan struct{}
	delayedCredentials        atomic.Bool
	waitForCredentials        chan struct{}
	delayedACLs               atomic.Bool
	waitForACLs               chan struct{}
	delay                     time.Duration
	r                         service.RequestHandle
}

func newTestRequestHandlerWithDelay(t *testing.T, dpsCfg service.Config, delay time.Duration) *testRequestHandlerWithDelay {
	return &testRequestHandlerWithDelay{
		RequestHandlerWithDps:     test.MakeRequestHandlerWithDps(t, dpsCfg),
		waitForTime:               make(chan struct{}),
		waitForOwnership:          make(chan struct{}),
		waitForCloudConfiguration: make(chan struct{}),
		waitForCredentials:        make(chan struct{}),
		waitForACLs:               make(chan struct{}),
		delay:                     delay,
	}
}

func (h *testRequestHandlerWithDelay) DefaultHandler(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	return h.r.DefaultHandler(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDelay) ProcessPlgdTime(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.delayedTime.CompareAndSwap(false, true) {
		time.Sleep(h.delay)
		close(h.waitForTime)
	}
	<-h.waitForTime
	return h.r.ProcessPlgdTime(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDelay) ProcessOwnership(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.delayedOwnership.CompareAndSwap(false, true) {
		time.Sleep(h.delay)
		close(h.waitForOwnership)
	}
	<-h.waitForOwnership
	return h.r.ProcessOwnership(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDelay) ProcessCloudConfiguration(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.delayedCloudConfiguration.CompareAndSwap(false, true) {
		time.Sleep(h.delay)
		close(h.waitForCloudConfiguration)
	}
	<-h.waitForCloudConfiguration
	return h.r.ProcessCloudConfiguration(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDelay) ProcessCredentials(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.delayedCredentials.CompareAndSwap(false, true) {
		time.Sleep(h.delay)
		close(h.waitForCredentials)
	}
	<-h.waitForCredentials
	return h.r.ProcessCredentials(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDelay) ProcessACLs(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.delayedACLs.CompareAndSwap(false, true) {
		time.Sleep(h.delay)
		close(h.waitForACLs)
	}
	<-h.waitForACLs
	return h.r.ProcessACLs(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDelay) Verify(ctx context.Context) error {
	logCounter := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("unexpected counters delayedTime=%v delayedOwnership=%v delayedCloudConfiguration=%v delayedCredentials=%v delayedACLs=%v",
				h.delayedTime.Load(),
				h.delayedOwnership.Load(),
				h.delayedCloudConfiguration.Load(),
				h.delayedCredentials.Load(),
				h.delayedACLs.Load())
		case <-time.After(time.Second):
			logCounter++
			if logCounter%3 == 0 {
				h.Logf("delayedTime=%v delayedOwnership=%v delayedCloudConfiguration=%v delayedCredentials=%v delayedACLs=%v",
					h.delayedTime.Load(),
					h.delayedOwnership.Load(),
					h.delayedCloudConfiguration.Load(),
					h.delayedCredentials.Load(),
					h.delayedACLs.Load())
			}
		}
		if h.delayedTime.Load() &&
			h.delayedOwnership.Load() &&
			h.delayedCloudConfiguration.Load() &&
			h.delayedCredentials.Load() &&
			h.delayedACLs.Load() {
			return nil
		}
	}
}

func TestProvisioningWithDelay(t *testing.T) {
	dpsCfg := test.MakeConfig(t)
	rh := newTestRequestHandlerWithDelay(t, dpsCfg, 10*time.Second)
	testProvisioningWithDPSHandler(t, rh, time.Minute*5)
}
