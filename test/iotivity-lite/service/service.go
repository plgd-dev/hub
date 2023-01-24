package service

import (
	"sync"
	"time"

	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/test/coap-gateway/test"
)

const (
	SignUpKey       = "SignUp"  // register
	SignOffKey      = "SignOff" // deregister
	SignInKey       = "SignIn"  // log in
	SignOutKey      = "SignOut" // log out
	PublishKey      = "Publish"
	UnpublishKey    = "Unpublish"
	RefreshTokenKey = "RefreshToken"
)

type CoapHandlerWithCounter struct {
	*coapgwTest.DefaultObserverHandler

	CallCounter struct {
		Data map[string]int
		Lock sync.Mutex
	}

	signedInChan  chan struct{}
	signedOffChan chan struct{}
	refreshChan   chan struct{}
}

func NewCoapHandlerWithCounter(atLifetime int64) *CoapHandlerWithCounter {
	dh := coapgwTest.MakeDefaultObserverHandler(atLifetime)
	return &CoapHandlerWithCounter{
		DefaultObserverHandler: &dh,
		CallCounter: struct {
			Data map[string]int
			Lock sync.Mutex
		}{
			Data: make(map[string]int),
		},

		signedInChan:  make(chan struct{}),
		signedOffChan: make(chan struct{}),
		refreshChan:   make(chan struct{}),
	}
}

func (ch *CoapHandlerWithCounter) SignUp(req coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error) {
	resp, err := ch.DefaultObserverHandler.SignUp(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignUpKey]++
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

func (ch *CoapHandlerWithCounter) SignOff() error {
	err := ch.DefaultObserverHandler.SignOff()
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignOffKey]++
	signOffCount, ok := ch.CallCounter.Data[SignOffKey]
	ch.CallCounter.Lock.Unlock()
	if ok && signOffCount == 1 {
		close(ch.signedOffChan)
	}
	return err
}

func (ch *CoapHandlerWithCounter) WaitForFirstSignOff(timeout time.Duration) bool {
	select {
	case <-ch.signedOffChan:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (ch *CoapHandlerWithCounter) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	resp, err := ch.DefaultObserverHandler.SignIn(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignInKey]++
	signInCount, ok := ch.CallCounter.Data[SignInKey]
	ch.CallCounter.Lock.Unlock()
	if ok && signInCount == 1 {
		close(ch.signedInChan)
	}
	return resp, err
}

func (ch *CoapHandlerWithCounter) WaitForFirstSignIn(timeout time.Duration) bool {
	select {
	case <-ch.signedInChan:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (ch *CoapHandlerWithCounter) SignOut(req coapgwService.CoapSignInReq) error {
	err := ch.DefaultObserverHandler.SignOut(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignOutKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *CoapHandlerWithCounter) PublishResources(req coapgwTestService.PublishRequest) error {
	err := ch.DefaultObserverHandler.PublishResources(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[PublishKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *CoapHandlerWithCounter) UnpublishResources(req coapgwTestService.UnpublishRequest) error {
	err := ch.DefaultObserverHandler.UnpublishResources(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[UnpublishKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *CoapHandlerWithCounter) RefreshToken(req coapgwService.CoapRefreshTokenReq) (coapgwService.CoapRefreshTokenResp, error) {
	resp, err := ch.DefaultObserverHandler.RefreshToken(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[RefreshTokenKey]++
	refreshCount, ok := ch.CallCounter.Data[RefreshTokenKey]
	ch.CallCounter.Lock.Unlock()
	if ok && refreshCount == 1 {
		close(ch.refreshChan)
	}
	return resp, err
}

func (ch *CoapHandlerWithCounter) WaitForFirstRefreshToken(timeout time.Duration) bool {
	select {
	case <-ch.refreshChan:
		return true
	case <-time.After(timeout):
		return false
	}
}
